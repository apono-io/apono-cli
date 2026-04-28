package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/spf13/cobra"
)

const (
	LauncherKindGUI = "GUI"
	LauncherKindTUI = "TUI"
)

type ConsumedBy string

const (
	ConsumedByOS ConsumedBy = "OS"
	ConsumedByAD ConsumedBy = "AD"
)

type LauncherClientModel struct {
	Id          string
	DisplayName string
	Kind        string
	Setup       string
	Invocation  string
}

type LauncherFetchResult struct {
	Launchers  []LauncherClientModel
	ConsumedBy ConsumedBy
}

type Launcher struct {
	FetchLaunchers   func(context.Context, *aponoapi.AponoClient, string) (*LauncherFetchResult, error)
	RunShell         func(*cobra.Command, string) (int, string, error)
	WrapInTerminal   func(string) string
	IsTTY            func() bool
	ChooseErrHandler func(*cobra.Command) ErrorHandler
}

func NewLauncher() *Launcher {
	return &Launcher{
		FetchLaunchers:   fetchLaunchers,
		RunShell:         runShell,
		WrapInTerminal:   wrapInTerminal,
		IsTTY:            isTTY,
		ChooseErrHandler: chooseErrorHandler,
	}
}

func (l *Launcher) LaunchSession(cobraCmd *cobra.Command, client *aponoapi.AponoClient, sessionId, clientId string) error {
	handler := l.ChooseErrHandler(cobraCmd)

	result, err := l.FetchLaunchers(cobraCmd.Context(), client, sessionId)
	if err != nil {
		_ = handler.Handle("Apono", fmt.Sprintf("Could not fetch session details: %v", err), "")
		return err
	}

	// In headless context Portal/Slack already gated on consumedBy before firing the URI;
	// in TTY we have no upstream gate, so block AD-consumed creds and tell the user to reset.
	if l.IsTTY() && result.ConsumedBy != "" && result.ConsumedBy != ConsumedByOS {
		msg := fmt.Sprintf("credentials for this session were already used elsewhere. reset them with `apono access reset-credentials %s` and try again.", sessionId)
		_ = handler.Handle("Apono", msg, "")
		return fmt.Errorf("%s", msg)
	}

	launcher, ok := findLauncher(result.Launchers, clientId)
	if !ok {
		msg := fmt.Sprintf("client %q is not supported for this session.\navailable: %s", clientId, availableIds(result.Launchers))
		_ = handler.Handle("Apono", msg, "")
		return fmt.Errorf("%s", msg)
	}

	combined := joinSetupAndInvocation(launcher.Setup, launcher.Invocation)

	switch launcher.Kind {
	case LauncherKindGUI:
		return l.execAndHandle(cobraCmd, combined, handler)

	case LauncherKindTUI:
		if l.IsTTY() {
			return l.execAndHandle(cobraCmd, combined, handler)
		}
		// Headless TUI needs a real terminal window; protocol-handler invocation has no stdin TTY.
		return l.execAndHandle(cobraCmd, l.WrapInTerminal(combined), handler)

	default:
		err := fmt.Errorf("unknown launcher kind %q for client %q", launcher.Kind, clientId)
		_ = handler.Handle("Apono", err.Error(), "")
		return err
	}
}

func (l *Launcher) execAndHandle(cobraCmd *cobra.Command, combined string, handler ErrorHandler) error {
	exitCode, stderr, err := l.RunShell(cobraCmd, combined)
	if err == nil && exitCode == 0 {
		return nil
	}

	if err != nil {
		_ = handler.Handle("Apono", fmt.Sprintf("Failed to launch: %v", err), stderr)
		return err
	}
	msg := fmt.Sprintf("Launcher exited with code %d.", exitCode)
	_ = handler.Handle("Apono", msg, stderr)
	return fmt.Errorf("launcher exited with code %d", exitCode)
}

func findLauncher(launchers []LauncherClientModel, id string) (LauncherClientModel, bool) {
	for _, l := range launchers {
		if l.Id == id {
			return l, true
		}
	}
	return LauncherClientModel{}, false
}

func availableIds(launchers []LauncherClientModel) string {
	ids := make([]string, 0, len(launchers))
	for _, l := range launchers {
		ids = append(ids, l.Id)
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return "(none)"
	}
	return strings.Join(ids, ", ")
}

func joinSetupAndInvocation(setup, invocation string) string {
	setup = strings.TrimSpace(setup)
	invocation = strings.TrimSpace(invocation)
	switch {
	case setup == "" && invocation == "":
		return ""
	case setup == "":
		return invocation
	case invocation == "":
		return setup
	default:
		return setup + " && " + invocation
	}
}

func runShell(cobraCmd *cobra.Command, combined string) (int, string, error) {
	if strings.TrimSpace(combined) == "" {
		return 0, "", fmt.Errorf("empty launcher command")
	}

	var stderr bytes.Buffer
	cmd := exec.CommandContext(cobraCmd.Context(), "sh", "-c", combined)
	cmd.Stdout = cobraCmd.OutOrStdout()
	cmd.Stdin = cobraCmd.InOrStdin()
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), stderr.String(), nil
		}
		return -1, stderr.String(), err
	}
	return 0, stderr.String(), nil
}

// TODO(DVL-8799): replace Terminal.app with iTerm2 detection + fallback.
func wrapInTerminal(combined string) string {
	escaped := strings.ReplaceAll(combined, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return fmt.Sprintf(`osascript -e 'tell application "Terminal" to do script "%s"' -e 'tell application "Terminal" to activate'`, escaped)
}

func isTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// TODO(DVL-8786): replace stub with real call once the BE endpoint and clientapi regen land.
func fetchLaunchers(ctx context.Context, client *aponoapi.AponoClient, sessionId string) (*LauncherFetchResult, error) {
	return stubLaunchersForSession(ctx, client, sessionId)
}
