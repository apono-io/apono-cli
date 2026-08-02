package connect

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/logshipping"
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

type shippedEntry struct {
	level   string
	message string
	fields  map[string]string
}

func captureReports(s *ClientStarter) *[]shippedEntry {
	var entries []shippedEntry
	s.Report = func(_ context.Context, level, message string, fields map[string]string) {
		entries = append(entries, shippedEntry{level: level, message: message, fields: fields})
	}
	return &entries
}

func findEntry(entries []shippedEntry, message string) *shippedEntry {
	for i := range entries {
		if entries[i].message == message {
			return &entries[i]
		}
	}
	return nil
}

func hasEntry(entries []shippedEntry, level, message string) bool {
	for _, e := range entries {
		if e.level == level && e.message == message {
			return true
		}
	}
	return false
}

func TestStart_prefetchedResult_doesNotFetchAgain(t *testing.T) {
	clients := []clientapi.LauncherClientModel{
		newClientModel("dbeaver", ClientKindGUI, "", "echo invoke"),
	}
	s, runs, _ := testClientStarter(true, clients, aponoapi.ConsumedByAponoCli, nil)

	fetchCalls := 0
	s.FetchClients = func(_ context.Context, _ *aponoapi.AponoClient, _ string) (*ClientFetchResult, error) {
		fetchCalls++
		return nil, errors.New("FetchClients must not be called when a prefetched result is supplied")
	}

	prefetched := &ClientFetchResult{Clients: clients, ConsumedBy: aponoapi.ConsumedByAponoCli}
	if err := s.Start(newCobraCmd(), nil, "sess-1", "dbeaver", prefetched); err != nil {
		t.Fatalf("Start with prefetched result returned error: %v", err)
	}

	if fetchCalls != 0 {
		t.Errorf("expected no FetchClients calls when prefetched result supplied, got %d", fetchCalls)
	}
	if len(*runs) != 1 {
		t.Errorf("expected 1 runShell call (invocation) from the prefetched launcher, got %d", len(*runs))
	}
}

func TestStart_nilPrefetched_fetchesOnce(t *testing.T) {
	clients := []clientapi.LauncherClientModel{
		newClientModel("dbeaver", ClientKindGUI, "", "echo invoke"),
	}
	s, _, _ := testClientStarter(true, clients, aponoapi.ConsumedByAponoCli, nil)

	fetchCalls := 0
	s.FetchClients = func(_ context.Context, _ *aponoapi.AponoClient, _ string) (*ClientFetchResult, error) {
		fetchCalls++
		return &ClientFetchResult{Clients: clients, ConsumedBy: aponoapi.ConsumedByAponoCli}, nil
	}

	if err := s.Start(newCobraCmd(), nil, "sess-1", "dbeaver", nil); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if fetchCalls != 1 {
		t.Errorf("expected exactly 1 FetchClients call for the direct (nil prefetched) path, got %d", fetchCalls)
	}
}

func TestStart_failure_shipsRealErrorNotCannedMessage(t *testing.T) {
	clients := []clientapi.LauncherClientModel{
		newClientModel("dbeaver", ClientKindGUI, "", "false"),
	}
	s, _, _ := testClientStarter(true, clients, aponoapi.ConsumedByAponoCli, func() (int, string, error) {
		return 1, "boom on stderr", nil
	})
	entries := captureReports(s)

	if err := s.Start(newCobraCmd(), nil, "sess-1", "dbeaver", nil); err == nil {
		t.Fatal("expected error, got nil")
	}

	var errEntry *shippedEntry
	for i := range *entries {
		if (*entries)[i].level == logshipping.LevelError && (*entries)[i].message == "launcher: GUI launch failed" {
			errEntry = &(*entries)[i]
		}
	}
	if errEntry == nil {
		t.Fatalf("expected a shipped GUI-failure error entry, got %+v", *entries)
	}
	if !strings.Contains(errEntry.fields[fieldError], "boom on stderr") {
		t.Errorf("expected shipped error field to carry the real stderr, got %q", errEntry.fields[fieldError])
	}
	if errEntry.fields[fieldAccessSessionID] != "sess-1" || errEntry.fields[fieldClientID] != "dbeaver" {
		t.Errorf("expected session/client fields, got %+v", errEntry.fields)
	}
}

func TestStart_happyPath_shipsInfoStepsIncludingLaunched(t *testing.T) {
	clients := []clientapi.LauncherClientModel{
		newClientModel("dbeaver", ClientKindGUI, "", "echo invoke"),
	}
	s, _, _ := testClientStarter(true, clients, aponoapi.ConsumedByAponoCli, nil)
	entries := captureReports(s)

	if err := s.Start(newCobraCmd(), nil, "sess-1", "dbeaver", nil); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	for _, msg := range []string{
		"launcher: starting",
		"launcher: session details fetched",
		"launcher: client resolved",
		"launcher: launching client",
		"launcher: client launched",
	} {
		if !hasEntry(*entries, logshipping.LevelInfo, msg) {
			t.Errorf("expected INFO step %q to be shipped, entries: %+v", msg, *entries)
		}
	}
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

			err := s.Start(newCobraCmd(), nil, "sess-1", "dbeaver", nil)
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

			if err := s.Start(newCobraCmd(), nil, "sess-1", "k9s", nil); err != nil {
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

func TestStart_TUI_NoTTY_wrapsAuthAndInvocationTogether(t *testing.T) {
	for _, kind := range []string{ClientKindTUI, ClientKindTERMINAL, ClientKindCLI} {
		t.Run(kind, func(t *testing.T) {
			clients := []clientapi.LauncherClientModel{
				newClientModel("k9s", kind, "setup", "k9s"),
			}
			s, runs, wraps := testClientStarter(false, clients, aponoapi.ConsumedByAponoCli, nil)

			if err := s.Start(newCobraCmd(), nil, "sess-1", "k9s", nil); err != nil {
				t.Fatalf("Start returned error: %v", err)
			}

			if len(*wraps) != 1 {
				t.Fatalf("expected 1 buildTerminalLaunchCommand call, got %d", len(*wraps))
			}
			if (*wraps)[0] != "setup && k9s" {
				t.Errorf("expected wrap input to be auth && invocation, got %q", (*wraps)[0])
			}
			if len(*runs) != 1 {
				t.Fatalf("expected 1 runShell call (only the wrapped command, auth not run inline), got %d", len(*runs))
			}
			if !strings.HasPrefix((*runs)[0].combined, "WRAPPED(") {
				t.Errorf("%s without TTY should run wrapped command, got %q", kind, (*runs)[0].combined)
			}
		})
	}
}

func TestStart_TUI_NoTTY_wrapBuilderFails_returnsWrappedError(t *testing.T) {
	clients := []clientapi.LauncherClientModel{
		newClientModel("k9s", ClientKindTERMINAL, "setup", "k9s"),
	}
	s, runs, _ := testClientStarter(false, clients, aponoapi.ConsumedByAponoCli, nil)
	s.BuildTerminalLaunchCommand = func(string) (string, error) {
		return "", errors.New("osascript path missing")
	}

	err := s.Start(newCobraCmd(), nil, "sess-1", "k9s", nil)
	if err == nil {
		t.Fatal("expected error when BuildTerminalLaunchCommand fails, got nil")
	}
	if !strings.Contains(err.Error(), "build terminal launch command") {
		t.Errorf("expected wrapped error mentioning the failed step, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "osascript path missing") {
		t.Errorf("expected wrapped error to include the underlying cause, got %q", err.Error())
	}
	if len(*runs) != 0 {
		t.Errorf("expected no runShell calls when wrap builder fails, got %d", len(*runs))
	}
}

func TestStart_TUI_NoTTY_emptyAuth_wrapsInvocationOnly(t *testing.T) {
	clients := []clientapi.LauncherClientModel{
		newClientModel("k9s", ClientKindTERMINAL, "", "k9s"),
	}
	s, runs, wraps := testClientStarter(false, clients, aponoapi.ConsumedByAponoCli, nil)

	if err := s.Start(newCobraCmd(), nil, "sess-1", "k9s", nil); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	if len(*wraps) != 1 {
		t.Fatalf("expected 1 buildTerminalLaunchCommand call, got %d", len(*wraps))
	}
	if (*wraps)[0] != "k9s" {
		t.Errorf("expected wrap input to be invocation only when auth is empty, got %q", (*wraps)[0])
	}
	if len(*runs) != 1 {
		t.Fatalf("expected 1 runShell call, got %d", len(*runs))
	}
}

func TestStart_unknownClient_errorsWithAvailableList(t *testing.T) {
	clients := []clientapi.LauncherClientModel{
		newClientModel("dbeaver", ClientKindGUI, "", ""),
		newClientModel("tableplus", ClientKindGUI, "", ""),
		newClientModel("cli", ClientKindTUI, "", ""),
	}
	s, runs, _ := testClientStarter(true, clients, aponoapi.ConsumedByAponoCli, nil)

	err := s.Start(newCobraCmd(), nil, "sess-1", "nonexistent", nil)
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

	err := s.Start(newCobraCmd(), nil, "sess-1", "dbeaver", nil)
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

	err := s.Start(newCobraCmd(), nil, "sess-1", "dbeaver", nil)
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

	if err := s.Start(newCobraCmd(), nil, "sess-1", "dbeaver", nil); err != nil {
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

	if err := s.Start(newCobraCmd(), nil, "sess-1", "dbeaver", nil); err != nil {
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

	if err := s.Start(newCobraCmd(), nil, "sess-1", "tableplus", nil); err != nil {
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

func TestLaunchFailureLevel(t *testing.T) {
	cases := []struct {
		name  string
		cause error
		want  string
	}{
		{"no cause", nil, logshipping.LevelError},
		{
			name:  "shell could not find the binary",
			cause: errors.New("client exited with code 127\nsh: psql: command not found"),
			want:  logshipping.LevelWarn,
		},
		{
			name:  "shell could not reach a path inside the app bundle",
			cause: errors.New("client exited with code 1\nsh: line 11: cd: /Applications/pgAdmin 4.app/Contents/Resources/web: No such file or directory"),
			want:  logshipping.LevelWarn,
		},
		{
			name:  "open could not resolve the application",
			cause: errors.New("client exited with code 1\nUnable to find application named 'TablePlus'"),
			want:  logshipping.LevelWarn,
		},
		{
			name:  "the client ran and objected",
			cause: errors.New("client exited with code 1\npsql: error: connection to server at \"h\" failed: FATAL: role \"u\" does not exist"),
			want:  logshipping.LevelError,
		},
		{
			name:  "a domain-level miss is not an install problem",
			cause: errors.New("client exited with code 1\nmongosh: collection not found"),
			want:  logshipping.LevelError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := launchFailureLevel(tc.cause); got != tc.want {
				t.Errorf("launchFailureLevel() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestReporters_clearSecretsWithoutHelpFromTheCallSite(t *testing.T) {
	const secret = "p@ss w&rd!"

	cases := []struct {
		name   string
		report func(s *ClientStarter, cause error)
	}{
		{
			name: "reportLauncher",
			report: func(s *ClientStarter, cause error) {
				s.reportLauncher(context.Background(), logshipping.LevelError, "launcher: some step failed", cause, "sess-1", "psql", ClientKindGUI, true)
			},
		},
		{
			name: "reportCommandFailure",
			report: func(s *ClientStarter, cause error) {
				s.reportCommandFailure(context.Background(), logshipping.LevelError, "launcher: GUI launch failed", 1, cause, "sess-1", "psql", ClientKindGUI, true)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &ClientStarter{secrets: []string{secret}}
			entries := captureReports(s)

			tc.report(s, errors.New(`psql "postgres://u:`+secret+`@h/db" rejected`))

			if len(*entries) != 1 {
				t.Fatalf("expected 1 shipped entry, got %d", len(*entries))
			}
			shipped := (*entries)[0].fields[fieldError]
			if strings.Contains(shipped, secret) {
				t.Errorf("the reporter shipped the secret verbatim: %q", shipped)
			}
			if !strings.Contains(shipped, redactedMarker) {
				t.Errorf("expected the secret to be replaced, got %q", shipped)
			}
			if !strings.Contains(shipped, "rejected") {
				t.Errorf("expected the diagnostic around the secret to survive, got %q", shipped)
			}
		})
	}
}

func TestStart_launchFails_levelFollowsTheFailureText(t *testing.T) {
	cases := []struct {
		name       string
		kind       string
		tty        bool
		message    string
		stderr     string
		wantLevel  string
		wantClient string
	}{
		{
			name:       "GUI, application missing",
			kind:       ClientKindGUI,
			tty:        true,
			message:    "launcher: GUI launch failed",
			stderr:     "Unable to find application named 'TablePlus'",
			wantLevel:  logshipping.LevelWarn,
			wantClient: "tableplus",
		},
		{
			name:       "GUI, the application itself objected",
			kind:       ClientKindGUI,
			tty:        true,
			message:    "launcher: GUI launch failed",
			stderr:     "TablePlus: could not read connection profile",
			wantLevel:  logshipping.LevelError,
			wantClient: "tableplus",
		},
		{
			name:       "interactive, binary missing",
			kind:       ClientKindTERMINAL,
			tty:        true,
			message:    "launcher: interactive launch failed",
			stderr:     "sh: psql: command not found",
			wantLevel:  logshipping.LevelWarn,
			wantClient: "psql",
		},
		{
			name:       "interactive, the client itself objected",
			kind:       ClientKindTERMINAL,
			tty:        true,
			message:    "launcher: interactive launch failed",
			stderr:     "psql: error: FATAL: role \"u\" does not exist",
			wantLevel:  logshipping.LevelError,
			wantClient: "psql",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clients := []clientapi.LauncherClientModel{newClientModel(tc.wantClient, tc.kind, "", "run-it")}
			s, _, _ := testClientStarter(tc.tty, clients, aponoapi.ConsumedByAponoCli, func() (int, string, error) {
				return 1, tc.stderr, nil
			})
			entries := captureReports(s)

			if err := s.Start(newCobraCmd(), nil, "sess-1", tc.wantClient, nil); err == nil {
				t.Fatal("expected an error, got nil")
			}

			entry := findEntry(*entries, tc.message)
			if entry == nil {
				t.Fatalf("expected a %q entry, got %+v", tc.message, *entries)
			}
			if entry.level != tc.wantLevel {
				t.Errorf("level = %q, want %q for stderr %q", entry.level, tc.wantLevel, tc.stderr)
			}
		})
	}
}

func TestStart_launchFails_shippedTextHoldsNoPassword(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cacheDir := filepath.Join(home, ".apono", "cache")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "sess-1"), []byte(base64.StdEncoding.EncodeToString([]byte(passwordWithSpecials))), 0o600); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	encoded := encodePassword(passwordWithSpecials, passwordEncodingURL)

	tableplus := clientapi.LauncherClientModel{
		Id:                "tableplus",
		LauncherType:      ClientKindGUI,
		InvocationCommand: `open -a TablePlus "postgres://user:__APONO_PASSWORD__@host:5432/db"`,
		PasswordEncoding:  passwordEncodingURL,
	}
	clientStderr := "connection to postgres://user:" + encoded + "@host:5432/db failed, password " + passwordWithSpecials + " rejected"
	s, _, _ := testClientStarter(true, []clientapi.LauncherClientModel{tableplus}, aponoapi.ConsumedByAponoCli, func() (int, string, error) {
		return 2, clientStderr, nil
	})
	entries := captureReports(s)

	err := s.Start(newCobraCmd(), nil, "sess-1", "tableplus", nil)
	if err == nil {
		t.Fatal("expected error when the client fails, got nil")
	}
	if !strings.Contains(err.Error(), passwordWithSpecials) {
		t.Errorf("the user-facing error must keep the client output verbatim, got %q", err.Error())
	}

	launchEntry := findEntry(*entries, "launcher: GUI launch failed")
	if launchEntry == nil {
		t.Fatalf("expected a shipped launch-failure entry, got %+v", *entries)
	}
	if launchEntry.fields[fieldExitCode] != "2" {
		t.Errorf("expected shipped exit_code %q, got %q", "2", launchEntry.fields[fieldExitCode])
	}
	shipped := launchEntry.fields[fieldError]
	if !strings.Contains(shipped, "connection to postgres://user:") || !strings.Contains(shipped, "@host:5432/db failed") {
		t.Errorf("shipped text must keep the diagnostic around the redaction, got %q", shipped)
	}
	for _, form := range []string{passwordWithSpecials, encoded} {
		for key, value := range launchEntry.fields {
			if strings.Contains(value, form) {
				t.Errorf("field %q still carries the password form %q: %q", key, form, value)
			}
		}
	}
}

func TestStart_headlessLaunchFails_shippedTextHoldsNoPassword(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cacheDir := filepath.Join(home, ".apono", "cache")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "sess-1"), []byte(base64.StdEncoding.EncodeToString([]byte(passwordWithSpecials))), 0o600); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	encoded := encodePassword(passwordWithSpecials, passwordEncodingURL)

	psql := clientapi.LauncherClientModel{
		Id:                "psql",
		LauncherType:      ClientKindTERMINAL,
		InvocationCommand: `psql "postgres://user:__APONO_PASSWORD__@host/db"`,
		PasswordEncoding:  passwordEncodingURL,
	}
	s, _, wraps := testClientStarter(false, []clientapi.LauncherClientModel{psql}, aponoapi.ConsumedByAponoCli, nil)
	s.RunShellCommand = func(_ *cobra.Command, combined string) (int, string, error) {
		return 1, "sh: cannot execute " + combined, nil
	}
	entries := captureReports(s)

	err := s.Start(newCobraCmd(), nil, "sess-1", "psql", nil)
	if err == nil {
		t.Fatal("expected error when the headless launch fails, got nil")
	}
	if len(*wraps) != 1 || !strings.Contains((*wraps)[0], encoded) {
		t.Fatalf("expected the wrapper to receive the substituted command, got %+v", *wraps)
	}
	if !strings.Contains(err.Error(), encoded) {
		t.Errorf("the user-facing error must keep the command verbatim, got %q", err.Error())
	}

	launchEntry := findEntry(*entries, "launcher: headless launch failed")
	if launchEntry == nil {
		t.Fatalf("expected a shipped headless-failure entry, got %+v", *entries)
	}
	if launchEntry.fields[fieldExitCode] != "1" {
		t.Errorf("expected shipped exit_code %q, got %q", "1", launchEntry.fields[fieldExitCode])
	}
	if !strings.Contains(launchEntry.fields[fieldError], "sh: cannot execute") {
		t.Errorf("shipped text must keep the diagnostic, got %q", launchEntry.fields[fieldError])
	}
	for _, form := range []string{passwordWithSpecials, encoded} {
		for key, value := range launchEntry.fields {
			if strings.Contains(value, form) {
				t.Errorf("field %q still carries the password form %q: %q", key, form, value)
			}
		}
	}
}

func TestStart_wrapBuilderFails_shippedTextHoldsNoPassword(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cacheDir := filepath.Join(home, ".apono", "cache")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "sess-1"), []byte(base64.StdEncoding.EncodeToString([]byte(passwordWithSpecials))), 0o600); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	encoded := encodePassword(passwordWithSpecials, passwordEncodingURL)

	psql := clientapi.LauncherClientModel{
		Id:                "psql",
		LauncherType:      ClientKindTERMINAL,
		InvocationCommand: `psql "postgres://user:__APONO_PASSWORD__@host/db"`,
		PasswordEncoding:  passwordEncodingURL,
	}
	s, runs, _ := testClientStarter(false, []clientapi.LauncherClientModel{psql}, aponoapi.ConsumedByAponoCli, nil)
	s.BuildTerminalLaunchCommand = func(command string) (string, error) {
		return "", errors.New("write launch script for " + command + ": no space left on device")
	}
	entries := captureReports(s)

	err := s.Start(newCobraCmd(), nil, "sess-1", "psql", nil)
	if err == nil {
		t.Fatal("expected error when the wrap builder fails, got nil")
	}
	if len(*runs) != 0 {
		t.Errorf("expected no runShell calls when the wrap builder fails, got %d", len(*runs))
	}
	if !strings.Contains(err.Error(), encoded) {
		t.Errorf("the user-facing error must keep the command verbatim, got %q", err.Error())
	}

	wrapEntry := findEntry(*entries, "launcher: build terminal launch command failed")
	if wrapEntry == nil {
		t.Fatalf("expected a shipped wrap-failure entry, got %+v", *entries)
	}
	if !strings.Contains(wrapEntry.fields[fieldError], "no space left on device") {
		t.Errorf("shipped text must keep the diagnostic, got %q", wrapEntry.fields[fieldError])
	}
	for _, form := range []string{passwordWithSpecials, encoded} {
		for key, value := range wrapEntry.fields {
			if strings.Contains(value, form) {
				t.Errorf("field %q still carries the password form %q: %q", key, form, value)
			}
		}
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

	if err := s.Start(newCobraCmd(), nil, "sess-1", "dbeaver", nil); err != nil {
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

	err := s.Start(newCobraCmd(), nil, "sess-missing", "tableplus", nil)
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

	err := s.Start(newCobraCmd(), nil, "sess-1", "dbeaver", nil)
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

func TestStart_authFails_shipsStructuredFieldsOnly(t *testing.T) {
	const authOutput = "auth-cmd output marker"

	clients := []clientapi.LauncherClientModel{
		newClientModel("dbeaver", ClientKindGUI, "auth-cmd", "invocation-cmd"),
	}
	s, _, _ := testClientStarter(true, clients, aponoapi.ConsumedByAponoCli, func() (int, string, error) {
		return 7, authOutput, nil
	})
	entries := captureReports(s)

	err := s.Start(newCobraCmd(), nil, "sess-1", "dbeaver", nil)
	if err == nil {
		t.Fatal("expected error when auth_command fails, got nil")
	}
	if !strings.Contains(err.Error(), authOutput) {
		t.Errorf("the user-facing error must keep the command output, got %q", err.Error())
	}

	authEntry := findEntry(*entries, "launcher: auth command failed")
	if authEntry == nil {
		t.Fatalf("expected a shipped auth-failure entry, got %+v", *entries)
	}
	if authEntry.level != logshipping.LevelWarn {
		t.Errorf("expected WARN level, got %q", authEntry.level)
	}

	want := map[string]string{
		fieldAccessSessionID: "sess-1",
		fieldClientID:        "dbeaver",
		fieldLauncherType:    ClientKindGUI,
		fieldIsTerminal:      "true",
		fieldExitCode:        "7",
	}
	if !maps.Equal(authEntry.fields, want) {
		t.Errorf("shipped fields = %+v, want exactly %+v", authEntry.fields, want)
	}
}

func TestStart_authCommandNeverRan_shipsExitCodeMinusOne(t *testing.T) {
	clients := []clientapi.LauncherClientModel{
		newClientModel("dbeaver", ClientKindGUI, "auth-cmd", "invocation-cmd"),
	}
	s, _, _ := testClientStarter(true, clients, aponoapi.ConsumedByAponoCli, func() (int, string, error) {
		return -1, "", errors.New("fork/exec: resource temporarily unavailable")
	})
	entries := captureReports(s)

	if err := s.Start(newCobraCmd(), nil, "sess-1", "dbeaver", nil); err == nil {
		t.Fatal("expected error when the auth command cannot be spawned, got nil")
	}

	authEntry := findEntry(*entries, "launcher: auth command failed")
	if authEntry == nil {
		t.Fatalf("expected a shipped auth-failure entry, got %+v", *entries)
	}
	if authEntry.fields[fieldExitCode] != "-1" {
		t.Errorf("expected exit_code %q when the process never ran, got %q", "-1", authEntry.fields[fieldExitCode])
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

	if err := s.Start(newCobraCmd(), nil, "sess-1", "tableplus", nil); err != nil {
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
