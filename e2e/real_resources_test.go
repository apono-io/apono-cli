//go:build e2e

package e2e

import (
	"context"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/apono-io/apono-cli/e2e/mockapi"
)

// These tests connect to real resources started next to the test run (GitHub Actions service
// containers, a kind cluster). The mock backend hands the CLI a connect command that points at
// them, so "apono access use --run" really opens a connection. They are skipped when the
// resource is not configured:
//
//	E2E_PG_URL             postgres://user:password@localhost:5432/db
//	E2E_KIND_KUBECONFIG    admin kubeconfig of a disposable cluster (kind)
//	E2E_REQUIRE_RESOURCES  set to 1 in CI so a missing resource fails instead of skipping

const kubeContext = "apono-e2e"

func skipOrFail(t *testing.T, format string, args ...any) {
	t.Helper()
	if os.Getenv("E2E_REQUIRE_RESOURCES") == "1" {
		t.Fatalf(format, args...)
	}
	t.Skipf(format, args...)
}

func requireEnv(t *testing.T, key string) string {
	t.Helper()
	value := os.Getenv(key)
	if value == "" {
		skipOrFail(t, "%s not set", key)
	}
	return value
}

func requireTool(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		skipOrFail(t, "%s not installed", name)
	}
}

func TestRealConnect_Postgres(t *testing.T) {
	pgURL := requireEnv(t, "E2E_PG_URL")
	requireTool(t, "psql")

	parsed, err := url.Parse(pgURL)
	if err != nil {
		t.Fatalf("invalid E2E_PG_URL: %v", err)
	}

	cli, srv := newLoggedInCLI(t)
	srv.Update(func(st *mockapi.State) {
		st.Session(mockapi.PostgresSessionID).Details = mockapi.Details(`psql "` + pgURL + `" -tAc "select current_user"`)
	})

	result := cli.MustRun("access", "use", mockapi.PostgresSessionID, "--run")
	if got := strings.TrimSpace(result.Stdout); got != parsed.User.Username() {
		t.Fatalf("expected psql to connect as %q, got:\n%s", parsed.User.Username(), result)
	}
}

func TestRealConnect_PostgresBadCredentials(t *testing.T) {
	pgURL := requireEnv(t, "E2E_PG_URL")
	requireTool(t, "psql")

	parsed, err := url.Parse(pgURL)
	if err != nil {
		t.Fatalf("invalid E2E_PG_URL: %v", err)
	}
	parsed.User = url.UserPassword(parsed.User.Username(), "wrong-password")

	cli, srv := newLoggedInCLI(t)
	srv.Update(func(st *mockapi.State) {
		st.Session(mockapi.PostgresSessionID).Details = mockapi.Details(`psql "` + parsed.String() + `" -tAc "select 1"`)
	})

	result := cli.MustFail("password authentication failed", "access", "use", mockapi.PostgresSessionID, "--run")
	assertContains(t, result.Stdout, "Problem to connect? Reset credentials")
}

// TestRealConnect_KubernetesCreatesMissingContext covers the full kube flow: the backend first
// returns only "use-context", the context is missing from the (empty) kubeconfig, the CLI reissues
// credentials and runs the returned set-cluster/set-credentials/set-context command, and the
// resulting context can talk to the cluster.
func TestRealConnect_KubernetesCreatesMissingContext(t *testing.T) {
	adminKubeconfig := requireEnv(t, "E2E_KIND_KUBECONFIG")
	requireTool(t, "kubectl")

	server, caData, token := provisionServiceAccount(t, adminKubeconfig)

	cli, srv := newLoggedInCLI(t)
	writeEmptyKubeconfig(t, cli.KubeconfigPath())

	afterReset := mockapi.Details(strings.Join([]string{
		"kubectl config set-cluster " + kubeContext + " --server=" + server,
		"kubectl config set clusters." + kubeContext + ".certificate-authority-data " + caData,
		"kubectl config set-credentials " + kubeContext + " --token=" + token,
		"kubectl config set-context " + kubeContext + " --cluster=" + kubeContext + " --user=" + kubeContext,
		"kubectl config use-context " + kubeContext,
	}, " && "))
	srv.Update(func(st *mockapi.State) {
		session := st.Session(mockapi.KubeSessionID)
		session.Details = mockapi.Details("kubectl config use-context " + kubeContext)
		session.DetailsAfterReset = &afterReset
	})

	result := cli.MustRun("access", "use", mockapi.KubeSessionID, "--run")
	assertContains(t, result.Stdout, "reissuing credentials")

	kubeconfig := cli.KubeconfigPath()
	if got := kubectl(t, kubeconfig, "config", "current-context"); got != kubeContext {
		t.Fatalf("expected current context %q, got %q", kubeContext, got)
	}
	if got := kubectl(t, kubeconfig, "get", "namespace", "default", "-o", "name"); got != "namespace/default" {
		t.Fatalf("new context cannot read the cluster, got %q", got)
	}

	// Running again finds the context and must not reissue credentials a second time.
	cli.MustRun("access", "use", mockapi.KubeSessionID, "--run")
	if n := len(srv.CallsTo("POST /api/client/v1/access-sessions/" + mockapi.KubeSessionID + "/reset-credentials")); n != 1 {
		t.Fatalf("expected exactly 1 reset call, got %d", n)
	}
}

const serviceAccountManifest = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: apono-e2e
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: apono-e2e-view
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: view
subjects:
  - kind: ServiceAccount
    name: apono-e2e
    namespace: default
`

// provisionServiceAccount plays the part of the Apono connector: it creates a read-only service
// account and returns what a real backend would put in the connect command.
func provisionServiceAccount(t *testing.T, adminKubeconfig string) (server, caData, token string) {
	t.Helper()

	apply := exec.Command("kubectl", "--kubeconfig", adminKubeconfig, "apply", "-f", "-")
	apply.Stdin = strings.NewReader(serviceAccountManifest)
	if out, err := apply.CombinedOutput(); err != nil {
		t.Fatalf("create service account: %v\n%s", err, out)
	}

	server = kubectl(t, adminKubeconfig, "config", "view", "--raw", "--minify", "-o", "jsonpath={.clusters[0].cluster.server}")
	caData = kubectl(t, adminKubeconfig, "config", "view", "--raw", "--minify", "-o", "jsonpath={.clusters[0].cluster.certificate-authority-data}")
	token = kubectl(t, adminKubeconfig, "create", "token", "apono-e2e", "-n", "default", "--duration", "10m")

	return server, caData, token
}

func kubectl(t *testing.T, kubeconfig string, args ...string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kubectl", append([]string{"--kubeconfig", kubeconfig}, args...)...) //nolint:gosec // test helper, args come from the test code
	out, err := cmd.Output()
	if err != nil {
		var stderr string
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
		}
		t.Fatalf("kubectl %v: %v\n%s", args, err, stderr)
	}
	return strings.TrimSpace(string(out))
}

func writeEmptyKubeconfig(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("apiVersion: v1\nkind: Config\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}
