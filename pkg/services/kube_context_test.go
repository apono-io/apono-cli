package services

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
)

const (
	shortKubeCommand = "kubectl config use-context my-cluster"
	fullKubeCommand  = "kubectl config set-cluster my-cluster --server=https://k8s && kubectl config set-credentials my-user --token=abc && kubectl config set-context my-cluster --cluster=my-cluster --user=my-user && kubectl config use-context my-cluster"
)

func TestExtractKubeContext(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    string
		wantOK  bool
	}{
		{name: "plain", command: "kubectl config use-context my-cluster", want: "my-cluster", wantOK: true},
		{name: "double quoted", command: `kubectl config use-context "my-cluster"`, want: "my-cluster", wantOK: true},
		{name: "single quoted", command: `kubectl config use-context 'my-cluster'`, want: "my-cluster", wantOK: true},
		{name: "chained", command: "kubectl config use-context my-cluster && kubectl get pods", want: "my-cluster", wantOK: true},
		{name: "full setup", command: fullKubeCommand, want: "my-cluster", wantOK: true},
		{name: "eks arn", command: "kubectl config use-context arn:aws:eks:us-east-1:123456789012:cluster/prod", want: "arn:aws:eks:us-east-1:123456789012:cluster/prod", wantOK: true},
		{name: "not kubectl", command: "psql -h localhost -U user", wantOK: false},
		{name: "kubectl without use-context", command: "kubectl get pods", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractKubeContext(tt.command)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("extractKubeContext(%q) = (%q, %v), want (%q, %v)", tt.command, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

type kubeStubs struct {
	existsCalls int
	resetCalls  int
	fetchCalls  int
}

func stubKube(t *testing.T, exists bool, existsErr error, resetErr error, fetched string) *kubeStubs {
	t.Helper()
	stubs := &kubeStubs{}

	origExists, origReset, origFetch := kubeContextExists, resetCredentialsFn, fetchCliCommandFn
	t.Cleanup(func() {
		kubeContextExists, resetCredentialsFn, fetchCliCommandFn = origExists, origReset, origFetch
	})

	kubeContextExists = func(_ context.Context, _ string) (bool, error) {
		stubs.existsCalls++
		return exists, existsErr
	}
	resetCredentialsFn = func(_ context.Context, _ *aponoapi.AponoClient, _ string) error {
		stubs.resetCalls++
		return resetErr
	}
	fetchCliCommandFn = func(_ context.Context, _ *aponoapi.AponoClient, _ string) (string, error) {
		stubs.fetchCalls++
		return fetched, nil
	}

	return stubs
}

func kubeSession(withCredentials, canReset bool) *clientapi.AccessSessionClientModel {
	session := &clientapi.AccessSessionClientModel{Id: "session-id"}
	if withCredentials {
		session.Credentials = *clientapi.NewNullableAccessSessionClientModelCredentials(
			clientapi.NewAccessSessionClientModelCredentials("cred-id", "stale", canReset),
		)
	}
	return session
}

func newKubeTestCommand() (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	var out bytes.Buffer
	cmd.SetOut(&out)
	return cmd, &out
}

func TestResolveKubeCliCommand(t *testing.T) {
	tests := []struct {
		name           string
		command        string
		session        *clientapi.AccessSessionClientModel
		exists         bool
		existsErr      error
		resetErr       error
		fetched        string
		want           string
		wantErr        bool
		wantExistsCall bool
		wantResetCall  bool
	}{
		{name: "non kubectl command", command: "psql -h localhost", session: kubeSession(true, true), want: "psql -h localhost"},
		{name: "full setup command is untouched", command: fullKubeCommand, session: kubeSession(true, true), want: fullKubeCommand},
		{name: "context exists", command: shortKubeCommand, session: kubeSession(true, true), exists: true, want: shortKubeCommand, wantExistsCall: true},
		{name: "existence check fails", command: shortKubeCommand, session: kubeSession(true, true), existsErr: errors.New("kubectl not found"), want: shortKubeCommand, wantExistsCall: true},
		{name: "context missing, credentials reissued", command: shortKubeCommand, session: kubeSession(true, true), fetched: fullKubeCommand, want: fullKubeCommand, wantExistsCall: true, wantResetCall: true},
		{name: "context missing, reset not allowed", command: shortKubeCommand, session: kubeSession(true, false), wantErr: true, wantExistsCall: true},
		{name: "context missing, no credentials", command: shortKubeCommand, session: kubeSession(false, false), wantErr: true, wantExistsCall: true},
		{name: "context missing, reset fails", command: shortKubeCommand, session: kubeSession(true, true), resetErr: errors.New("boom"), wantErr: true, wantExistsCall: true, wantResetCall: true},
		{name: "context missing, refetched command still short", command: shortKubeCommand, session: kubeSession(true, true), fetched: shortKubeCommand, wantErr: true, wantExistsCall: true, wantResetCall: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubs := stubKube(t, tt.exists, tt.existsErr, tt.resetErr, tt.fetched)
			cmd, _ := newKubeTestCommand()

			got, err := resolveKubeCliCommand(cmd, nil, tt.session, tt.command)
			if (err != nil) != tt.wantErr {
				t.Fatalf("resolveKubeCliCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("resolveKubeCliCommand() = %q, want %q", got, tt.want)
			}
			if (stubs.existsCalls > 0) != tt.wantExistsCall {
				t.Errorf("existence check calls = %d, want called = %v", stubs.existsCalls, tt.wantExistsCall)
			}
			if (stubs.resetCalls > 0) != tt.wantResetCall {
				t.Errorf("reset calls = %d, want called = %v", stubs.resetCalls, tt.wantResetCall)
			}
		})
	}
}

func TestResolveKubeCliCommand_printsNoticeWhenCreatingContext(t *testing.T) {
	stubKube(t, false, nil, nil, fullKubeCommand)
	cmd, out := newKubeTestCommand()

	if _, err := resolveKubeCliCommand(cmd, nil, kubeSession(true, true), shortKubeCommand); err != nil {
		t.Fatalf("resolveKubeCliCommand() error = %v", err)
	}

	if !bytes.Contains(out.Bytes(), []byte("my-cluster")) || !bytes.Contains(out.Bytes(), []byte("creating it")) {
		t.Errorf("expected notice about creating the context, got %q", out.String())
	}
}
