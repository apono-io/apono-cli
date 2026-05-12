package connect

import (
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
)

type runShellCall struct {
	combined string
}

func newCobraCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	return cmd
}

func newClientModel(id, kind, setup, invocation string) clientapi.LauncherClientModel {
	c := clientapi.LauncherClientModel{
		Id:                id,
		LauncherType:      kind,
		InvocationCommand: invocation,
	}
	if setup != "" {
		c.AuthCommand = *clientapi.NewNullableString(&setup)
	}
	return c
}

func testClientStarter(tty bool, clients []clientapi.LauncherClientModel, consumedBy string, runShellResult func() (int, string, error)) (*ClientStarter, *[]runShellCall, *[]string) {
	var runCalls []runShellCall
	var wrapCalls []string

	s := &ClientStarter{
		FetchClients: func(_ context.Context, _ *aponoapi.AponoClient, _ string) (*ClientFetchResult, error) {
			return &ClientFetchResult{Clients: clients, ConsumedBy: consumedBy}, nil
		},
		RunShellCommand: func(_ *cobra.Command, combined string) (int, string, error) {
			runCalls = append(runCalls, runShellCall{combined: combined})
			if runShellResult != nil {
				return runShellResult()
			}
			return 0, "", nil
		},
		BuildTerminalLaunchCommand: func(command string) (string, error) {
			wrapCalls = append(wrapCalls, command)
			return "WRAPPED(" + command + ")", nil
		},
		IsRunningInTerminal: func() bool { return tty },
	}

	return s, &runCalls, &wrapCalls
}

func TestStart_GUI_runsShellInline_regardlessOfTTY(t *testing.T) {
	cases := []struct {
		name string
		tty  bool
	}{
		{"with-tty", true},
		{"without-tty", false},
	}

	clients := []clientapi.LauncherClientModel{
		newClientModel("dbeaver", ClientKindGUI, "echo setup", "echo invoke"),
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, runs, wraps := testClientStarter(tc.tty, clients, aponoapi.ConsumedByAponoCli, nil)

			err := s.Start(newCobraCmd(), nil, "sess-1", "dbeaver")
			if err != nil {
				t.Fatalf("Start returned error: %v", err)
			}

			if len(*runs) != 2 {
				t.Fatalf("expected 2 runShell calls (auth + invocation), got %d", len(*runs))
			}
			if (*runs)[0].combined != "echo setup" {
				t.Errorf("expected first call to be auth_command, got %q", (*runs)[0].combined)
			}
			if (*runs)[1].combined != "echo invoke" {
				t.Errorf("expected second call to be invocation_command, got %q", (*runs)[1].combined)
			}
			if len(*wraps) != 0 {
				t.Errorf("GUI should never wrap in Terminal, got %d wrap calls", len(*wraps))
			}
		})
	}
}

func TestStart_TUI_TTY_runsInline(t *testing.T) {
	for _, kind := range []string{ClientKindTUI, ClientKindTERMINAL, ClientKindCLI} {
		t.Run(kind, func(t *testing.T) {
			clients := []clientapi.LauncherClientModel{
				newClientModel("k9s", kind, "setup", "k9s"),
			}
			s, runs, wraps := testClientStarter(true, clients, aponoapi.ConsumedByAponoCli, nil)

			if err := s.Start(newCobraCmd(), nil, "sess-1", "k9s"); err != nil {
				t.Fatalf("Start returned error: %v", err)
			}

			if len(*runs) != 2 {
				t.Fatalf("expected 2 runShell calls (auth + invocation), got %d", len(*runs))
			}
			if len(*wraps) != 0 {
				t.Errorf("%s with TTY should not wrap, got %d wrap calls", kind, len(*wraps))
			}
			for i, call := range *runs {
				if strings.HasPrefix(call.combined, "WRAPPED(") {
					t.Errorf("%s with TTY should run inline, got wrapped command at index %d: %q", kind, i, call.combined)
				}
			}
		})
	}
}

func TestStart_TUI_NoTTY_wrapsInTerminal(t *testing.T) {
	for _, kind := range []string{ClientKindTUI, ClientKindTERMINAL, ClientKindCLI} {
		t.Run(kind, func(t *testing.T) {
			clients := []clientapi.LauncherClientModel{
				newClientModel("k9s", kind, "setup", "k9s"),
			}
			s, runs, wraps := testClientStarter(false, clients, aponoapi.ConsumedByAponoCli, nil)

			if err := s.Start(newCobraCmd(), nil, "sess-1", "k9s"); err != nil {
				t.Fatalf("Start returned error: %v", err)
			}

			// Auth runs inline (no wrap), invocation gets wrapped.
			if len(*wraps) != 1 {
				t.Fatalf("expected 1 buildTerminalLaunchCommand call, got %d", len(*wraps))
			}
			if len(*runs) != 2 {
				t.Fatalf("expected 2 runShell calls (auth inline + wrapped invocation), got %d", len(*runs))
			}
			if (*runs)[0].combined != "setup" {
				t.Errorf("expected first call to be plain auth_command, got %q", (*runs)[0].combined)
			}
			if !strings.HasPrefix((*runs)[1].combined, "WRAPPED(") {
				t.Errorf("%s without TTY should wrap invocation, got %q", kind, (*runs)[1].combined)
			}
		})
	}
}

func TestStart_unknownClient_errorsWithAvailableList(t *testing.T) {
	clients := []clientapi.LauncherClientModel{
		newClientModel("dbeaver", ClientKindGUI, "", ""),
		newClientModel("tableplus", ClientKindGUI, "", ""),
		newClientModel("cli", ClientKindTUI, "", ""),
	}
	s, runs, _ := testClientStarter(true, clients, aponoapi.ConsumedByAponoCli, nil)

	err := s.Start(newCobraCmd(), nil, "sess-1", "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown client, got nil")
	}
	if len(*runs) != 0 {
		t.Errorf("expected no runShell calls on bad client id, got %d", len(*runs))
	}
	if !strings.Contains(err.Error(), "Supported clients") {
		t.Errorf("expected error to list available clients, got %q", err.Error())
	}
	for _, want := range []string{"cli", "dbeaver", "tableplus"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected error to mention %q, got %q", want, err.Error())
		}
	}
}

func TestStart_shellNonZeroExit_returnsErrorWithStderr(t *testing.T) {
	clients := []clientapi.LauncherClientModel{
		newClientModel("dbeaver", ClientKindGUI, "true", "false"),
	}
	s, runs, _ := testClientStarter(true, clients, aponoapi.ConsumedByAponoCli, func() (int, string, error) {
		return 1, "boom on stderr", nil
	})

	err := s.Start(newCobraCmd(), nil, "sess-1", "dbeaver")
	if err == nil {
		t.Fatal("expected error on non-zero exit, got nil")
	}
	if len(*runs) != 1 {
		t.Fatalf("expected 1 runShell call, got %d", len(*runs))
	}
	if !strings.Contains(err.Error(), "boom on stderr") {
		t.Errorf("expected error to surface stderr tail, got %q", err.Error())
	}
}

func TestStart_TTY_consumedByOther_blocks(t *testing.T) {
	clients := []clientapi.LauncherClientModel{
		newClientModel("dbeaver", ClientKindGUI, "s", "i"),
	}
	s, runs, _ := testClientStarter(true, clients, "someone-else", nil)

	err := s.Start(newCobraCmd(), nil, "sess-1", "dbeaver")
	if err == nil {
		t.Fatal("expected error when creds consumed elsewhere in TTY context, got nil")
	}
	if len(*runs) != 0 {
		t.Errorf("expected no runShell calls when blocked on consumedBy, got %d", len(*runs))
	}
	if !strings.Contains(err.Error(), "reset") {
		t.Errorf("expected error to mention reset, got %q", err.Error())
	}
}

func TestStart_NoTTY_consumedByOther_proceeds(t *testing.T) {
	// Headless context: Portal/Slack already gated upstream, CLI trusts and proceeds.
	clients := []clientapi.LauncherClientModel{
		newClientModel("dbeaver", ClientKindGUI, "", "i"),
	}
	s, runs, _ := testClientStarter(false, clients, "someone-else", nil)

	if err := s.Start(newCobraCmd(), nil, "sess-1", "dbeaver"); err != nil {
		t.Fatalf("expected success in headless context regardless of consumedBy, got %v", err)
	}
	if len(*runs) != 1 {
		t.Errorf("expected 1 runShell call in headless context, got %d", len(*runs))
	}
}

func TestStart_TTY_consumedByEmpty_proceeds(t *testing.T) {
	clients := []clientapi.LauncherClientModel{
		newClientModel("dbeaver", ClientKindGUI, "", "i"),
	}
	s, runs, _ := testClientStarter(true, clients, "", nil)

	if err := s.Start(newCobraCmd(), nil, "sess-1", "dbeaver"); err != nil {
		t.Fatalf("expected success when consumedBy is empty (fresh session), got %v", err)
	}
	if len(*runs) != 1 {
		t.Errorf("expected 1 runShell call, got %d", len(*runs))
	}
}

func TestAvailableIDs_emptyList(t *testing.T) {
	if got := availableIDs(nil); got != "(none)" {
		t.Errorf("expected '(none)' for empty list, got %q", got)
	}
}

func TestAvailableIDs_sorted(t *testing.T) {
	got := availableIDs([]clientapi.LauncherClientModel{
		newClientModel("tableplus", "", "", ""),
		newClientModel("dbeaver", "", "", ""),
		newClientModel("cli", "", "", ""),
	})
	want := "cli, dbeaver, tableplus"
	if got != want {
		t.Errorf("availableIDs() = %q, want %q", got, want)
	}
}

func TestStart_substitutesPasswordPlaceholder_withURLEncoding(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cacheDir := filepath.Join(home, ".apono", "cache")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}
	rawPwd := passwordWithSpecials
	if err := os.WriteFile(filepath.Join(cacheDir, "sess-1"), []byte(base64.StdEncoding.EncodeToString([]byte(rawPwd))), 0o600); err != nil {
		t.Fatalf("write cache: %v", err)
	}

	tableplus := clientapi.LauncherClientModel{
		Id:                "tableplus",
		LauncherType:      ClientKindGUI,
		InvocationCommand: `open -a TablePlus "postgres://user:__APONO_PASSWORD__@host:5432/db"`,
		PasswordEncoding:  "url",
	}
	s, runs, _ := testClientStarter(true, []clientapi.LauncherClientModel{tableplus}, aponoapi.ConsumedByAponoCli, nil)

	if err := s.Start(newCobraCmd(), nil, "sess-1", "tableplus"); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if len(*runs) != 1 {
		t.Fatalf("expected 1 runShell call, got %d", len(*runs))
	}
	if strings.Contains((*runs)[0].combined, "__APONO_PASSWORD__") {
		t.Errorf("placeholder not substituted, got %q", (*runs)[0].combined)
	}
	wantEncoded := `p%40ss+w%26rd%21`
	if !strings.Contains((*runs)[0].combined, wantEncoded) {
		t.Errorf("expected url-encoded password %q in command, got %q", wantEncoded, (*runs)[0].combined)
	}
}

func TestStart_noPlaceholder_skipsCacheRead(t *testing.T) {
	// HOME points at an empty temp dir — if the substitution path were hit,
	// readCachedPassword would error. dbeaver's invocation has no placeholder,
	// so the cache must not be read.
	t.Setenv("HOME", t.TempDir())

	clients := []clientapi.LauncherClientModel{
		newClientModel("dbeaver", ClientKindGUI, "echo setup", `dbeaver -con "host=h|password=$(base64 -d -i ~/.apono/cache/sess-1)"`),
	}
	s, runs, _ := testClientStarter(true, clients, aponoapi.ConsumedByAponoCli, nil)

	if err := s.Start(newCobraCmd(), nil, "sess-1", "dbeaver"); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if len(*runs) != 2 {
		t.Fatalf("expected 2 runShell calls (auth + invocation), got %d", len(*runs))
	}
}

func TestStart_placeholderButCacheMissing_returnsError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	tableplus := clientapi.LauncherClientModel{
		Id:                "tableplus",
		LauncherType:      ClientKindGUI,
		InvocationCommand: `open -a TablePlus "postgres://user:__APONO_PASSWORD__@host/db"`,
		PasswordEncoding:  "url",
	}
	s, runs, _ := testClientStarter(true, []clientapi.LauncherClientModel{tableplus}, aponoapi.ConsumedByAponoCli, nil)

	err := s.Start(newCobraCmd(), nil, "sess-missing", "tableplus")
	if err == nil {
		t.Fatal("expected error when cache file missing, got nil")
	}
	if len(*runs) != 0 {
		t.Errorf("expected no runShell calls when cache missing, got %d", len(*runs))
	}
	if !strings.Contains(err.Error(), "resolve credentials") {
		t.Errorf("expected error to mention credential resolution, got %q", err.Error())
	}
}

func TestStart_authFails_invocationSkipped(t *testing.T) {
	clients := []clientapi.LauncherClientModel{
		newClientModel("dbeaver", ClientKindGUI, "auth-cmd", "invocation-cmd"),
	}
	calls := 0
	s, runs, _ := testClientStarter(true, clients, aponoapi.ConsumedByAponoCli, func() (int, string, error) {
		calls++
		if calls == 1 {
			return 1, "auth boom", nil
		}
		return 0, "", nil
	})

	err := s.Start(newCobraCmd(), nil, "sess-1", "dbeaver")
	if err == nil {
		t.Fatal("expected error when auth_command fails, got nil")
	}
	if len(*runs) != 1 {
		t.Fatalf("expected only the auth runShell call before bailing, got %d", len(*runs))
	}
	if (*runs)[0].combined != "auth-cmd" {
		t.Errorf("expected first call to be auth_command, got %q", (*runs)[0].combined)
	}
}

func TestStart_authFirstThenSubstitution(t *testing.T) {
	// Auth runs first, then placeholder substitution reads the cache file.
	// This test wires a runShellResult that creates the cache file as a side
	// effect of the auth call — proving the ordering: substitution sees the
	// fresh cache populated by auth, not a stale or missing one.
	home := t.TempDir()
	t.Setenv("HOME", home)
	cacheDir := filepath.Join(home, ".apono", "cache")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}

	tableplus := newClientModel("tableplus", ClientKindGUI, "populate-cache", `open "postgres://u:__APONO_PASSWORD__@h/d"`)
	tableplus.PasswordEncoding = "url"

	calls := 0
	s, runs, _ := testClientStarter(true, []clientapi.LauncherClientModel{tableplus}, aponoapi.ConsumedByAponoCli, func() (int, string, error) {
		calls++
		if calls == 1 {
			// Simulate auth populating the cache.
			_ = os.WriteFile(filepath.Join(cacheDir, "sess-1"), []byte(base64.StdEncoding.EncodeToString([]byte("hello"))), 0o600)
		}
		return 0, "", nil
	})

	if err := s.Start(newCobraCmd(), nil, "sess-1", "tableplus"); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if len(*runs) != 2 {
		t.Fatalf("expected 2 runShell calls (auth then invocation), got %d", len(*runs))
	}
	if (*runs)[0].combined != "populate-cache" {
		t.Errorf("expected first call to be auth_command, got %q", (*runs)[0].combined)
	}
	if !strings.Contains((*runs)[1].combined, "hello") {
		t.Errorf("expected invocation to have substituted password from cache populated by auth, got %q", (*runs)[1].combined)
	}
}
