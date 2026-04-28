package services

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/spf13/cobra"
)

type capturingErrorHandler struct {
	calls []handledError
}

type handledError struct {
	title      string
	message    string
	stderrTail string
}

func (c *capturingErrorHandler) Handle(title, message, stderrTail string) error {
	c.calls = append(c.calls, handledError{title, message, stderrTail})
	return nil
}

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

func testLauncher(tty bool, launchers []LauncherClientModel, consumedBy ConsumedBy, runShellResult func() (int, string, error)) (*Launcher, *[]runShellCall, *[]string, *capturingErrorHandler) {
	var runCalls []runShellCall
	var wrapCalls []string
	errorHandler := &capturingErrorHandler{}

	l := &Launcher{
		FetchLaunchers: func(_ context.Context, _ *aponoapi.AponoClient, _ string) (*LauncherFetchResult, error) {
			return &LauncherFetchResult{Launchers: launchers, ConsumedBy: consumedBy}, nil
		},
		RunShell: func(_ *cobra.Command, combined string) (int, string, error) {
			runCalls = append(runCalls, runShellCall{combined: combined})
			if runShellResult != nil {
				return runShellResult()
			}
			return 0, "", nil
		},
		WrapInTerminal: func(combined string) string {
			wrapCalls = append(wrapCalls, combined)
			return "WRAPPED(" + combined + ")"
		},
		IsTTY:            func() bool { return tty },
		ChooseErrHandler: func(_ *cobra.Command) ErrorHandler { return errorHandler },
	}

	return l, &runCalls, &wrapCalls, errorHandler
}

func TestLaunchSession_GUI_runsShellInline_regardlessOfTTY(t *testing.T) {
	cases := []struct {
		name string
		tty  bool
	}{
		{"with-tty", true},
		{"without-tty", false},
	}

	launchers := []LauncherClientModel{
		{Id: "dbeaver", Kind: LauncherKindGUI, Setup: "echo setup", Invocation: "echo invoke"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l, runs, wraps, _ := testLauncher(tc.tty, launchers, ConsumedByOS, nil)

			err := l.LaunchSession(newCobraCmd(), nil, "sess-1", "dbeaver")
			if err != nil {
				t.Fatalf("LaunchSession returned error: %v", err)
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

func TestLaunchSession_TUI_TTY_runsInline(t *testing.T) {
	launchers := []LauncherClientModel{
		{Id: "k9s", Kind: LauncherKindTUI, Setup: "setup", Invocation: "k9s"},
	}
	l, runs, wraps, _ := testLauncher(true, launchers, ConsumedByOS, nil)

	if err := l.LaunchSession(newCobraCmd(), nil, "sess-1", "k9s"); err != nil {
		t.Fatalf("LaunchSession returned error: %v", err)
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

func TestLaunchSession_TUI_NoTTY_wrapsInTerminal(t *testing.T) {
	launchers := []LauncherClientModel{
		{Id: "k9s", Kind: LauncherKindTUI, Setup: "setup", Invocation: "k9s"},
	}
	l, runs, wraps, _ := testLauncher(false, launchers, ConsumedByOS, nil)

	if err := l.LaunchSession(newCobraCmd(), nil, "sess-1", "k9s"); err != nil {
		t.Fatalf("LaunchSession returned error: %v", err)
	}

	if len(*wraps) != 1 {
		t.Fatalf("expected 1 wrapInTerminal call, got %d", len(*wraps))
	}
	if len(*runs) != 1 {
		t.Fatalf("expected 1 runShell call, got %d", len(*runs))
	}
	if !strings.HasPrefix((*runs)[0].combined, "WRAPPED(") {
		t.Errorf("TUI without TTY should run wrapped command, got %q", (*runs)[0].combined)
	}
}

func TestLaunchSession_unknownClient_errorsWithAvailableList(t *testing.T) {
	launchers := []LauncherClientModel{
		{Id: "dbeaver", Kind: LauncherKindGUI},
		{Id: "tableplus", Kind: LauncherKindGUI},
		{Id: "cli", Kind: LauncherKindTUI},
	}
	l, runs, _, errorHandler := testLauncher(true, launchers, ConsumedByOS, nil)

	err := l.LaunchSession(newCobraCmd(), nil, "sess-1", "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown client, got nil")
	}
	if len(*runs) != 0 {
		t.Errorf("expected no runShell calls on bad client id, got %d", len(*runs))
	}
	if !strings.Contains(err.Error(), "available:") {
		t.Errorf("expected error to list available clients, got %q", err.Error())
	}
	for _, want := range []string{"cli", "dbeaver", "tableplus"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected error to mention %q, got %q", want, err.Error())
		}
	}
	if len(errorHandler.calls) != 1 {
		t.Errorf("expected 1 Handle() call for unknown client, got %d", len(errorHandler.calls))
	}
}

func TestLaunchSession_shellNonZeroExit_reportsAndReturnsError(t *testing.T) {
	launchers := []LauncherClientModel{
		{Id: "dbeaver", Kind: LauncherKindGUI, Setup: "true", Invocation: "false"},
	}
	l, runs, _, errorHandler := testLauncher(true, launchers, ConsumedByOS, func() (int, string, error) {
		return 1, "boom on stderr", nil
	})

	err := l.LaunchSession(newCobraCmd(), nil, "sess-1", "dbeaver")
	if err == nil {
		t.Fatal("expected error on non-zero exit, got nil")
	}
	if len(*runs) != 1 {
		t.Fatalf("expected 1 runShell call, got %d", len(*runs))
	}
	if len(errorHandler.calls) != 1 {
		t.Fatalf("expected 1 Handle() call on shell failure, got %d", len(errorHandler.calls))
	}
	if errorHandler.calls[0].stderrTail != "boom on stderr" {
		t.Errorf("expected stderr tail to be propagated to Handle(), got %q", errorHandler.calls[0].stderrTail)
	}
}

func TestLaunchSession_TTY_consumedByAD_blocks(t *testing.T) {
	launchers := []LauncherClientModel{
		{Id: "dbeaver", Kind: LauncherKindGUI, Setup: "s", Invocation: "i"},
	}
	l, runs, _, errorHandler := testLauncher(true, launchers, ConsumedByAD, nil)

	err := l.LaunchSession(newCobraCmd(), nil, "sess-1", "dbeaver")
	if err == nil {
		t.Fatal("expected error when consumedBy=AD in TTY context, got nil")
	}
	if len(*runs) != 0 {
		t.Errorf("expected no runShell calls when blocked on consumedBy, got %d", len(*runs))
	}
	if !strings.Contains(err.Error(), "reset") {
		t.Errorf("expected error to mention reset, got %q", err.Error())
	}
	if len(errorHandler.calls) != 1 {
		t.Errorf("expected 1 Handle() call when blocked, got %d", len(errorHandler.calls))
	}
}

func TestLaunchSession_NoTTY_consumedByAD_proceeds(t *testing.T) {
	// Headless context: Portal/Slack already gated upstream, CLI trusts and proceeds.
	launchers := []LauncherClientModel{
		{Id: "dbeaver", Kind: LauncherKindGUI, Setup: "s", Invocation: "i"},
	}
	l, runs, _, _ := testLauncher(false, launchers, ConsumedByAD, nil)

	if err := l.LaunchSession(newCobraCmd(), nil, "sess-1", "dbeaver"); err != nil {
		t.Fatalf("expected success in headless context regardless of consumedBy, got %v", err)
	}
	if len(*runs) != 1 {
		t.Errorf("expected 1 runShell call in headless context, got %d", len(*runs))
	}
}

func TestLaunchSession_TTY_consumedByEmpty_proceeds(t *testing.T) {
	launchers := []LauncherClientModel{
		{Id: "dbeaver", Kind: LauncherKindGUI, Setup: "s", Invocation: "i"},
	}
	l, runs, _, _ := testLauncher(true, launchers, "", nil)

	if err := l.LaunchSession(newCobraCmd(), nil, "sess-1", "dbeaver"); err != nil {
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
		got := joinSetupAndInvocation(tc.setup, tc.invocation)
		if got != tc.want {
			t.Errorf("joinSetupAndInvocation(%q, %q) = %q, want %q", tc.setup, tc.invocation, got, tc.want)
		}
	}
}

func TestAvailableIDs_emptyList(t *testing.T) {
	if got := availableIds(nil); got != "(none)" {
		t.Errorf("expected '(none)' for empty list, got %q", got)
	}
}

func TestAvailableIDs_sorted(t *testing.T) {
	got := availableIds([]LauncherClientModel{
		{Id: "tableplus"}, {Id: "dbeaver"}, {Id: "cli"},
	})
	want := "cli, dbeaver, tableplus"
	if got != want {
		t.Errorf("availableIds() = %q, want %q", got, want)
	}
}

func TestWrapInTerminal_escapesQuotesAndBackslashes(t *testing.T) {
	got := wrapInTerminal(`echo "hi" \n`)
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
