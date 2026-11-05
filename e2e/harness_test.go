//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/apono-io/apono-cli/e2e/mockapi"
)

const commandTimeout = 90 * time.Second

// Result is the outcome of one CLI invocation.
type Result struct {
	Args     []string
	Stdout   string
	Stderr   string
	ExitCode int
}

func (r Result) String() string {
	return "apono " + strings.Join(r.Args, " ") +
		"\nexit code: " + strconv.Itoa(r.ExitCode) +
		"\n--- stdout ---\n" + r.Stdout +
		"\n--- stderr ---\n" + r.Stderr
}

// CLI runs the apono binary with an isolated HOME, config dir and KUBECONFIG.
type CLI struct {
	t     *testing.T
	Home  string
	env   map[string]string
	Stdin string
}

func newCLI(t *testing.T) *CLI {
	t.Helper()

	home := t.TempDir()
	env := map[string]string{
		"HOME":            home,
		"XDG_CONFIG_HOME": filepath.Join(home, ".config"),
		"KUBECONFIG":      filepath.Join(home, ".kube", "config"),
		"PATH":            os.Getenv("PATH"),
		"TMPDIR":          os.TempDir(),
		"TERM":            "dumb",
		"NO_COLOR":        "1",
	}
	if coverDir := os.Getenv("E2E_COVERDIR"); coverDir != "" {
		env["GOCOVERDIR"] = coverDir
	}

	return &CLI{t: t, Home: home, env: env}
}

// newLoggedInCLI starts a mock backend, logs in to it and fails the test if the CLI calls a
// route the mock does not implement.
func newLoggedInCLI(t *testing.T) (*CLI, *mockapi.Server) {
	t.Helper()

	srv := mockapi.New(t)
	cli := newCLI(t)
	cli.Login(srv, "default")
	t.Cleanup(func() { assertNoUnexpectedCalls(t, srv) })

	return cli, srv
}

// SetEnv sets an environment variable for subsequent runs.
func (c *CLI) SetEnv(key, value string) { c.env[key] = value }

// PrependPath puts dir first on PATH so its executables shadow real tools.
func (c *CLI) PrependPath(dir string) {
	c.env["PATH"] = dir + string(os.PathListSeparator) + c.env["PATH"]
}

// KubeconfigPath is the isolated kubeconfig the CLI and kubectl use.
func (c *CLI) KubeconfigPath() string { return c.env["KUBECONFIG"] }

// Run executes apono with args and never fails the test by itself.
func (c *CLI) Run(args ...string) Result {
	c.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath, args...)
	cmd.Dir = c.Home
	cmd.Env = c.environ()
	cmd.Stdin = strings.NewReader(c.Stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := Result{Args: args, Stdout: stdout.String(), Stderr: stderr.String()}

	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		result.ExitCode = exitErr.ExitCode()
	default:
		c.t.Fatalf("failed to run apono %v: %v", args, err)
	}

	if ctx.Err() != nil {
		c.t.Fatalf("apono %v timed out after %s\n%s", args, commandTimeout, result)
	}

	return result
}

// MustRun executes apono and fails the test unless it exits 0.
func (c *CLI) MustRun(args ...string) Result {
	c.t.Helper()

	result := c.Run(args...)
	if result.ExitCode != 0 {
		c.t.Fatalf("expected success, got:\n%s", result)
	}
	return result
}

// MustFail executes apono and fails the test unless it exits non-zero with stderr containing wantErr.
func (c *CLI) MustFail(wantErr string, args ...string) Result {
	c.t.Helper()

	result := c.Run(args...)
	if result.ExitCode == 0 {
		c.t.Fatalf("expected failure, got success:\n%s", result)
	}
	if !strings.Contains(result.Stderr, wantErr) {
		c.t.Fatalf("expected stderr to contain %q, got:\n%s", wantErr, result)
	}
	return result
}

// Login runs a non-interactive personal-token login against the mock.
func (c *CLI) Login(srv *mockapi.Server, profile string) Result {
	c.t.Helper()

	return c.MustRun("login",
		"--profile", profile,
		"--personal-token", srv.Token(),
		"--api-url", srv.URL,
		"--app-url", srv.URL,
		"--portal-url", srv.URL,
	)
}

func (c *CLI) environ() []string {
	env := make([]string, 0, len(c.env))
	for k, v := range c.env {
		env = append(env, k+"="+v)
	}
	return env
}

// decodeJSON parses CLI JSON output into T.
func decodeJSON[T any](t *testing.T, result Result) T {
	t.Helper()

	var v T
	if err := json.Unmarshal([]byte(result.Stdout), &v); err != nil {
		t.Fatalf("stdout is not valid JSON for %T: %v\n%s", v, err, result)
	}
	return v
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected output to contain %q, got:\n%s", needle, haystack)
	}
}

func assertNotContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Fatalf("expected output not to contain %q, got:\n%s", needle, haystack)
	}
}

// assertNoUnexpectedCalls fails if the CLI called a route the mock does not implement, which
// usually means the CLI started using a new endpoint that the mock (and this suite) must cover.
func assertNoUnexpectedCalls(t *testing.T, srv *mockapi.Server) {
	t.Helper()
	for _, call := range srv.Unexpected() {
		t.Errorf("CLI called an endpoint the mock does not implement: %s %s?%s", call.Method, call.Path, call.Query)
	}
}
