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

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/utils"
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

type LauncherFetchResult struct {
	Launchers  []clientapi.LauncherClientModel
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
		ChooseErrHandler: ChooseErrorHandler,
	}
}

func (l *Launcher) LaunchSession(cobraCmd *cobra.Command, client *aponoapi.AponoClient, sessionID, clientID string) error {
	errorHandler := l.ChooseErrHandler(cobraCmd)

	result, err := l.FetchLaunchers(cobraCmd.Context(), client, sessionID)
	if err != nil {
		_ = errorHandler.Handle(defaultErrorTitle, fmt.Sprintf("Could not fetch session details: %v", err), "")
		return err
	}

	// In headless context Portal/Slack already gated on consumedBy before firing the URI;
	// in TTY we have no upstream gate, so block AD-consumed creds and tell the user to reset.
	if l.IsTTY() && result.ConsumedBy != "" && result.ConsumedBy != ConsumedByOS {
		msg := fmt.Sprintf("credentials for this session were already used elsewhere. reset them with `apono access reset-credentials %s` and try again.", sessionID)
		_ = errorHandler.Handle(defaultErrorTitle, msg, "")
		return fmt.Errorf("%s", msg)
	}

	launcher, ok := findLauncher(result.Launchers, clientID)
	if !ok {
		msg := fmt.Sprintf("client %q is not supported for this session.\navailable: %s", clientID, availableIDs(result.Launchers))
		_ = errorHandler.Handle(defaultErrorTitle, msg, "")
		return fmt.Errorf("%s", msg)
	}

	combined := joinSetupAndInvocation(utils.FromNullableString(launcher.AuthCommand), launcher.InvocationCommand)

	switch launcher.LauncherType {
	case LauncherKindGUI:
		return l.execAndHandle(cobraCmd, combined, errorHandler)

	case LauncherKindTUI:
		if l.IsTTY() {
			return l.execAndHandle(cobraCmd, combined, errorHandler)
		}
		// Headless TUI needs a real terminal window; .app/AppleScript invocation has no stdin TTY.
		return l.execAndHandle(cobraCmd, l.WrapInTerminal(combined), errorHandler)

	default:
		err := fmt.Errorf("unknown launcher kind %q for client %q", launcher.LauncherType, clientID)
		_ = errorHandler.Handle(defaultErrorTitle, err.Error(), "")
		return err
	}
}

func (l *Launcher) execAndHandle(cobraCmd *cobra.Command, combined string, errorHandler ErrorHandler) error {
	exitCode, stderr, err := l.RunShell(cobraCmd, combined)
	if err == nil && exitCode == 0 {
		return nil
	}

	if err != nil {
		_ = errorHandler.Handle(defaultErrorTitle, fmt.Sprintf("Failed to launch: %v", err), stderr)
		return err
	}
	msg := fmt.Sprintf("Launcher exited with code %d.", exitCode)
	_ = errorHandler.Handle(defaultErrorTitle, msg, stderr)
	return fmt.Errorf("launcher exited with code %d", exitCode)
}

func findLauncher(launchers []clientapi.LauncherClientModel, id string) (clientapi.LauncherClientModel, bool) {
	for _, l := range launchers {
		if l.Id == id {
			return l, true
		}
	}
	return clientapi.LauncherClientModel{}, false
}

func availableIDs(launchers []clientapi.LauncherClientModel) string {
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

func fetchLaunchers(ctx context.Context, client *aponoapi.AponoClient, sessionID string) (*LauncherFetchResult, error) {
	details, _, err := client.ClientAPI.AccessSessionsAPI.
		GetAccessSessionAccessDetails(ctx, sessionID).
		ConsumedBy(string(ConsumedByOS)).
		Execute()
	if err != nil {
		return nil, err
	}
	return &LauncherFetchResult{
		Launchers:  details.Launchers,
		ConsumedBy: ConsumedBy(utils.FromNullableString(details.ConsumedBy)),
	}, nil
}
