//go:build e2e

package e2e

import (
	"testing"

	"github.com/apono-io/apono-cli/e2e/mockapi"
)

// These tests cover "apono access use <id> --run" with stub client binaries: the CLI fetches the
// connect command from the backend and runs it with sh -c. See real_resources_test.go for the
// same flow against real Postgres and Kubernetes.

const kubeFullCommand = "kubectl config set-cluster e2e --server=https://127.0.0.1:6443 && " +
	"kubectl config set-credentials e2e --token=e2e-token && " +
	"kubectl config set-context e2e --cluster=e2e --user=e2e && " +
	"kubectl config use-context e2e"

const resetRoute = "POST /api/client/v1/access-sessions/" + mockapi.KubeSessionID + "/reset-credentials"

func TestConnect_Postgres(t *testing.T) {
	cli, _ := newLoggedInCLI(t)
	stubs := installStubs(t, cli, "psql")

	result := cli.MustRun("access", "use", mockapi.PostgresSessionID, "--run")

	assertContains(t, result.Stdout, "stub psql ran")
	stubs.AssertInvoked("psql postgresql://apono_e2e:e2e-password@localhost:5432/e2e -c select current_user")
}

func TestConnect_CommandFailureSuggestsReset(t *testing.T) {
	cli, _ := newLoggedInCLI(t)
	installStubs(t, cli, "psql")
	cli.SetEnv("E2E_STUB_EXIT", "2")

	result := cli.MustFail("error executing command", "access", "use", mockapi.PostgresSessionID, "--run")
	assertContains(t, result.Stdout, "Problem to connect? Reset credentials using this command")
	assertContains(t, result.Stdout, mockapi.PostgresSessionID)
}

func TestConnect_SessionWithoutCliMethod(t *testing.T) {
	cli, srv := newLoggedInCLI(t)
	srv.Update(func(st *mockapi.State) {
		st.Session(mockapi.PostgresSessionID).Model.ConnectionMethods = []string{"instructions"}
	})

	cli.MustFail("does not support cli access", "access", "use", mockapi.PostgresSessionID, "--run")
}

func TestConnect_Kube_ContextExists(t *testing.T) {
	cli, srv := newLoggedInCLI(t)
	stubs := installStubs(t, cli, "kubectl")
	cli.SetEnv("E2E_KUBE_CONTEXTS", "other e2e")

	cli.MustRun("access", "use", mockapi.KubeSessionID, "--run")

	stubs.AssertInvoked("kubectl config use-context e2e")
	if n := len(srv.CallsTo(resetRoute)); n != 0 {
		t.Fatalf("existing context must not reissue credentials, got %d reset calls", n)
	}
}

func TestConnect_Kube_ContextMissingReissuesCredentials(t *testing.T) {
	cli, srv := newLoggedInCLI(t)
	stubs := installStubs(t, cli, "kubectl")
	cli.SetEnv("E2E_KUBE_CONTEXTS", "other")
	afterReset := mockapi.Details(kubeFullCommand)
	srv.Update(func(st *mockapi.State) { st.Session(mockapi.KubeSessionID).DetailsAfterReset = &afterReset })

	result := cli.MustRun("access", "use", mockapi.KubeSessionID, "--run")

	assertContains(t, result.Stdout, "Kubernetes context e2e not found in your kubeconfig, reissuing credentials")
	if n := len(srv.CallsTo(resetRoute)); n != 1 {
		t.Fatalf("expected 1 reset call, got %d", n)
	}
	stubs.AssertInvoked("kubectl config set-cluster e2e --server=https://127.0.0.1:6443")
	stubs.AssertInvoked("kubectl config use-context e2e")
}

func TestConnect_Kube_ContextMissingCannotReset(t *testing.T) {
	cli, srv := newLoggedInCLI(t)
	installStubs(t, cli, "kubectl")
	srv.Update(func(st *mockapi.State) { st.Session(mockapi.KubeSessionID).Model.Credentials.Get().CanReset = false })

	cli.MustFail(`kubernetes context "e2e" does not exist in your kubeconfig and the credentials for session `+
		mockapi.KubeSessionID+" cannot be reissued", "access", "use", mockapi.KubeSessionID, "--run")
}

func TestConnect_Kube_ResetDoesNotReturnContext(t *testing.T) {
	cli, srv := newLoggedInCLI(t)
	installStubs(t, cli, "kubectl")

	cli.MustFail("new credentials were not returned for session "+mockapi.KubeSessionID,
		"access", "use", mockapi.KubeSessionID, "--run")
	if n := len(srv.CallsTo(resetRoute)); n != 1 {
		t.Fatalf("expected 1 reset call, got %d", n)
	}
}
