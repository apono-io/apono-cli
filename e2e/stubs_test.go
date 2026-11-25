//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// stubScript records its argv (one invocation per line) to $E2E_STUB_LOG, prints a marker on
// stdout and exits with $E2E_STUB_EXIT (default 0). kubectl's "config get-contexts" prints
// $E2E_KUBE_CONTEXTS so tests can choose whether a context already exists.
const stubScript = `#!/bin/sh
name=$(basename "$0")
{
  printf '%s' "$name"
  for arg in "$@"; do printf ' %s' "$arg"; done
  printf '\n'
} >> "$E2E_STUB_LOG"
if [ "$name" = "kubectl" ] && [ "$1" = "config" ] && [ "$2" = "get-contexts" ]; then
  for ctx in $E2E_KUBE_CONTEXTS; do echo "$ctx"; done
  exit 0
fi
echo "stub $name ran"
exit "${E2E_STUB_EXIT:-0}"
`

// Stubs is a directory of fake client binaries put first on the CLI's PATH.
type Stubs struct {
	t   *testing.T
	log string
}

// installStubs shadows the named tools (psql, kubectl, ...) with recording stubs.
func installStubs(t *testing.T, cli *CLI, tools ...string) *Stubs {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("stub binaries are shell scripts")
	}

	dir := t.TempDir()
	for _, tool := range tools {
		if err := os.WriteFile(filepath.Join(dir, tool), []byte(stubScript), 0o700); err != nil { //nolint:gosec // test stub must be executable
			t.Fatalf("write stub %s: %v", tool, err)
		}
	}

	logPath := filepath.Join(dir, "invocations.log")
	cli.PrependPath(dir)
	cli.SetEnv("E2E_STUB_LOG", logPath)

	return &Stubs{t: t, log: logPath}
}

// Invocations returns each recorded command line, e.g. "psql postgresql://... -c select 1".
func (s *Stubs) Invocations() []string {
	s.t.Helper()

	data, err := os.ReadFile(s.log)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		s.t.Fatalf("read stub log: %v", err)
	}
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

// AssertInvoked fails unless some invocation equals want.
func (s *Stubs) AssertInvoked(want string) {
	s.t.Helper()

	invocations := s.Invocations()
	for _, got := range invocations {
		if got == want {
			return
		}
	}
	s.t.Fatalf("expected stub invocation %q, got:\n%s", want, strings.Join(invocations, "\n"))
}
