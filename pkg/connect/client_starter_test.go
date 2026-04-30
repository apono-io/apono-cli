package connect

import (
	"bytes"
	"context"
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
		BuildTerminalLaunchCommand: func(command string) string {
			wrapCalls = append(wrapCalls, command)
			return "WRAPPED(" + command + ")"
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
			s, runs, wraps := testClientStarter(tc.tty, clients, consumedByAponoCli, nil)

			err := s.Start(newCobraCmd(), nil, "sess-1", "dbeaver")
			if err != nil {
				t.Fatalf("Start returned error: %v", err)
			}

			if len(*runs) != 1 {
				t.Fatalf("expected 1 runShell call, got %d", len(*runs))
			}
			if !strings.Contains((*runs)[0].combined, "echo setup") || !strings.Contains((*runs)[0].combined, "echo invoke") {
				t.Errorf("expected combined string to include setup and invocation, got %q", (*runs)[0].combined)
			}
			if !strings.Contains((*runs)[0].combined, "&&") {
				t.Errorf("expected setup and invocation joined with &&, got %q", (*runs)[0].combined)
			}
			if len(*wraps) != 0 {
				t.Errorf("GUI should never wrap in Terminal, got %d wrap calls", len(*wraps))
			}
		})
	}
}

func TestStart_TUI_TTY_runsInline(t *testing.T) {
	clients := []clientapi.LauncherClientModel{
		newClientModel("k9s", ClientKindTUI, "setup", "k9s"),
	}
	s, runs, wraps := testClientStarter(true, clients, consumedByAponoCli, nil)

	if err := s.Start(newCobraCmd(), nil, "sess-1", "k9s"); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	if len(*runs) != 1 {
		t.Fatalf("expected 1 runShell call, got %d", len(*runs))
	}
	if len(*wraps) != 0 {
		t.Errorf("TUI with TTY should not wrap, got %d wrap calls", len(*wraps))
	}
	if strings.HasPrefix((*runs)[0].combined, "WRAPPED(") {
		t.Errorf("TUI with TTY should run inline, got wrapped command %q", (*runs)[0].combined)
	}
}

func TestStart_TUI_NoTTY_wrapsInTerminal(t *testing.T) {
	clients := []clientapi.LauncherClientModel{
		newClientModel("k9s", ClientKindTUI, "setup", "k9s"),
	}
	s, runs, wraps := testClientStarter(false, clients, consumedByAponoCli, nil)

	if err := s.Start(newCobraCmd(), nil, "sess-1", "k9s"); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	if len(*wraps) != 1 {
		t.Fatalf("expected 1 buildTerminalLaunchCommand call, got %d", len(*wraps))
	}
	if len(*runs) != 1 {
		t.Fatalf("expected 1 runShell call, got %d", len(*runs))
	}
	if !strings.HasPrefix((*runs)[0].combined, "WRAPPED(") {
		t.Errorf("TUI without TTY should run wrapped command, got %q", (*runs)[0].combined)
	}
}

func TestStart_unknownClient_errorsWithAvailableList(t *testing.T) {
	clients := []clientapi.LauncherClientModel{
		newClientModel("dbeaver", ClientKindGUI, "", ""),
		newClientModel("tableplus", ClientKindGUI, "", ""),
		newClientModel("cli", ClientKindTUI, "", ""),
	}
	s, runs, _ := testClientStarter(true, clients, consumedByAponoCli, nil)

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
	s, runs, _ := testClientStarter(true, clients, consumedByAponoCli, func() (int, string, error) {
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
		newClientModel("dbeaver", ClientKindGUI, "s", "i"),
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
		newClientModel("dbeaver", ClientKindGUI, "s", "i"),
	}
	s, runs, _ := testClientStarter(true, clients, "", nil)

	if err := s.Start(newCobraCmd(), nil, "sess-1", "dbeaver"); err != nil {
		t.Fatalf("expected success when consumedBy is empty (fresh session), got %v", err)
	}
	if len(*runs) != 1 {
		t.Errorf("expected 1 runShell call, got %d", len(*runs))
	}
}

func TestJoinSetupAndInvocation(t *testing.T) {
	cases := []struct {
		setup, invocation, want string
	}{
		{"a", "b", "a && b"},
		{"  a  ", "  b  ", "a && b"},
		{"", "b", "b"},
		{"a", "", "a"},
		{"", "", ""},
	}
	for _, tc := range cases {
		got := combineSetupAndInvocationCommands(tc.setup, tc.invocation)
		if got != tc.want {
			t.Errorf("combineSetupAndInvocationCommands(%q, %q) = %q, want %q", tc.setup, tc.invocation, got, tc.want)
		}
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

func TestBuildTerminalLaunchCommand_escapesQuotesAndBackslashes(t *testing.T) {
	got := buildTerminalLaunchCommand(`echo "hi" \n`)
	if !strings.Contains(got, `\"hi\"`) {
		t.Errorf("expected double quotes to be escaped, got %q", got)
	}
	if !strings.Contains(got, `\\n`) {
		t.Errorf("expected backslashes to be escaped, got %q", got)
	}
	if !strings.Contains(got, `osascript`) || !strings.Contains(got, `Terminal`) {
		t.Errorf("expected osascript Terminal call, got %q", got)
	}
}

func TestBuildTerminalLaunchCommand_escapesSingleQuotes(t *testing.T) {
	// Outer wrapping is `sh -c '...'`, so a raw ' inside the command would close
	// the shell single-quoted string early. Each ' from the input must turn into
	// the shell close-escape-reopen sequence '\''.
	got := buildTerminalLaunchCommand(`echo 'hi'`)

	if !strings.Contains(got, `'\''hi'\''`) {
		t.Errorf("expected single quotes to be shell-escaped, got %q", got)
	}
}
