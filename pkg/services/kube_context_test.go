package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

type kubeStubConfig struct {
	exists    bool
	existsErr error
	resetErr  error
	fetched   string
	fetchErr  error
}

func stubKube(cfg kubeStubConfig) (kubeCommandResolver, *kubeStubs) {
	stubs := &kubeStubs{}
	resolver := kubeCommandResolver{
		contextExists: func(_ context.Context, _ string) (bool, error) {
			stubs.existsCalls++
			return cfg.exists, cfg.existsErr
		},
		resetCredentials: func(_ context.Context, _ *aponoapi.AponoClient, _ string) error {
			stubs.resetCalls++
			return cfg.resetErr
		},
		fetchCliCommand: func(_ context.Context, _ *aponoapi.AponoClient, _ string) (string, error) {
			stubs.fetchCalls++
			return cfg.fetched, cfg.fetchErr
		},
	}

	return resolver, stubs
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
	const otherContextCommand = "kubectl config set-context other-cluster --cluster=other && kubectl config use-context my-cluster"

	tests := []struct {
		name           string
		command        string
		session        *clientapi.AccessSessionClientModel
		stub           kubeStubConfig
		want           string
		wantErr        bool
		wantExistsCall bool
		wantResetCall  bool
		wantFetchCall  bool
	}{
		{name: "non kubectl command", command: "psql -h localhost", session: kubeSession(true, true), want: "psql -h localhost"},
		{name: "full setup command is untouched", command: fullKubeCommand, session: kubeSession(true, true), want: fullKubeCommand},
		{name: "set-context for another context is not a full setup", command: otherContextCommand, session: kubeSession(true, true), stub: kubeStubConfig{exists: true}, want: otherContextCommand, wantExistsCall: true},
		{name: "context exists", command: shortKubeCommand, session: kubeSession(true, true), stub: kubeStubConfig{exists: true}, want: shortKubeCommand, wantExistsCall: true},
		{name: "existence check fails", command: shortKubeCommand, session: kubeSession(true, true), stub: kubeStubConfig{existsErr: errors.New("kubectl not found")}, want: shortKubeCommand, wantExistsCall: true},
		{name: "context missing, credentials reissued", command: shortKubeCommand, session: kubeSession(true, true), stub: kubeStubConfig{fetched: fullKubeCommand}, want: fullKubeCommand, wantExistsCall: true, wantResetCall: true, wantFetchCall: true},
		{name: "context missing, reset not allowed", command: shortKubeCommand, session: kubeSession(true, false), wantErr: true, wantExistsCall: true},
		{name: "context missing, no credentials", command: shortKubeCommand, session: kubeSession(false, false), wantErr: true, wantExistsCall: true},
		{name: "context missing, reset fails", command: shortKubeCommand, session: kubeSession(true, true), stub: kubeStubConfig{resetErr: errors.New("boom")}, wantErr: true, wantExistsCall: true, wantResetCall: true},
		{name: "context missing, refetch fails", command: shortKubeCommand, session: kubeSession(true, true), stub: kubeStubConfig{fetchErr: errors.New("boom")}, wantErr: true, wantExistsCall: true, wantResetCall: true, wantFetchCall: true},
		{name: "context missing, refetched command still short", command: shortKubeCommand, session: kubeSession(true, true), stub: kubeStubConfig{fetched: shortKubeCommand}, wantErr: true, wantExistsCall: true, wantResetCall: true, wantFetchCall: true},
		{name: "context missing, refetched command sets another context", command: shortKubeCommand, session: kubeSession(true, true), stub: kubeStubConfig{fetched: otherContextCommand}, wantErr: true, wantExistsCall: true, wantResetCall: true, wantFetchCall: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resolver, stubs := stubKube(tt.stub)
			cmd, _ := newKubeTestCommand()

			got, err := resolver.resolve(cmd, nil, tt.session, tt.command)
			if (err != nil) != tt.wantErr {
				t.Fatalf("resolve() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("resolve() = %q, want %q", got, tt.want)
			}
			if (stubs.existsCalls > 0) != tt.wantExistsCall {
				t.Errorf("existence check calls = %d, want called = %v", stubs.existsCalls, tt.wantExistsCall)
			}
			if (stubs.resetCalls > 0) != tt.wantResetCall {
				t.Errorf("reset calls = %d, want called = %v", stubs.resetCalls, tt.wantResetCall)
			}
			if (stubs.fetchCalls > 0) != tt.wantFetchCall {
				t.Errorf("fetch calls = %d, want called = %v", stubs.fetchCalls, tt.wantFetchCall)
			}
		})
	}
}

func TestResolveKubeCliCommand_printsNoticeWhenCreatingContext(t *testing.T) {
	resolver, _ := stubKube(kubeStubConfig{fetched: fullKubeCommand})
	cmd, out := newKubeTestCommand()

	if _, err := resolver.resolve(cmd, nil, kubeSession(true, true), shortKubeCommand); err != nil {
		t.Fatalf("resolve() error = %v", err)
	}

	if !bytes.Contains(out.Bytes(), []byte("my-cluster")) || !bytes.Contains(out.Bytes(), []byte("creating it")) {
		t.Errorf("expected notice about creating the context, got %q", out.String())
	}
}

func TestSetsKubeContext(t *testing.T) {
	tests := []struct {
		name    string
		command string
		context string
		want    bool
	}{
		{name: "full setup", command: fullKubeCommand, context: "my-cluster", want: true},
		{name: "use-context only", command: shortKubeCommand, context: "my-cluster", want: false},
		{name: "quoted", command: `kubectl config set-context "my-cluster" --cluster=c`, context: "my-cluster", want: true},
		{name: "different context", command: "kubectl config set-context other --cluster=c", context: "my-cluster", want: false},
		{name: "prefix of another context", command: "kubectl config set-context my-cluster-2 --cluster=c", context: "my-cluster", want: false},
		{name: "current context flag", command: "kubectl config set-context --current --namespace=x", context: "my-cluster", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := setsKubeContext(tt.command, tt.context); got != tt.want {
				t.Errorf("setsKubeContext(%q, %q) = %v, want %v", tt.command, tt.context, got, tt.want)
			}
		})
	}
}

// fakeKubectl puts a kubectl script on PATH that exits with the given code.
func fakeKubectl(t *testing.T, exitCode int) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake kubectl script requires a unix shell")
	}

	dir := t.TempDir()
	script := fmt.Sprintf("#!/bin/sh\nexit %d\n", exitCode)
	if err := os.WriteFile(filepath.Join(dir, "kubectl"), []byte(script), 0o755); err != nil { //nolint:gosec // test executable
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func TestKubectlContextExists(t *testing.T) {
	t.Run("context found", func(t *testing.T) {
		fakeKubectl(t, 0)
		exists, err := kubectlContextExists(context.Background(), "my-cluster")
		if err != nil || !exists {
			t.Errorf("kubectlContextExists() = (%v, %v), want (true, nil)", exists, err)
		}
	})

	t.Run("context not found", func(t *testing.T) {
		fakeKubectl(t, 1)
		exists, err := kubectlContextExists(context.Background(), "my-cluster")
		if err != nil || exists {
			t.Errorf("kubectlContextExists() = (%v, %v), want (false, nil)", exists, err)
		}
	})

	t.Run("kubectl not installed", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		exists, err := kubectlContextExists(context.Background(), "my-cluster")
		if err == nil || exists {
			t.Errorf("kubectlContextExists() = (%v, %v), want (false, error)", exists, err)
		}
	})
}
