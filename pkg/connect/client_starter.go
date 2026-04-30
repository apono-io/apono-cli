package connect

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
	ClientKindGUI = "GUI"
	ClientKindTUI = "TUI"
)

type ClientStarter struct {
	FetchClients               func(context.Context, *aponoapi.AponoClient, string) (*ClientFetchResult, error)
	RunShellCommand            func(*cobra.Command, string) (int, string, error)
	BuildTerminalLaunchCommand func(string) string
	IsRunningInTerminal        func() bool
}

func NewClientStarter() *ClientStarter {
	return &ClientStarter{
		FetchClients:               fetchClients,
		RunShellCommand:            runShellCommand,
		BuildTerminalLaunchCommand: buildTerminalLaunchCommand,
		IsRunningInTerminal:        isRunningInTerminal,
	}
}

func (s *ClientStarter) Start(cobraCmd *cobra.Command, apiClient *aponoapi.AponoClient, sessionID, clientID string) error {
	result, err := s.FetchClients(cobraCmd.Context(), apiClient, sessionID)
	if err != nil {
		return fmt.Errorf("could not fetch session details: %w", err)
	}

	// Portal and Slack show their own "credentials already in use" prompt before
	// firing the apono:// URI, so a headless (executed from protocol handler) run can trust that. A terminal user typed
	// the command directly and never saw that prompt - surface it here ourselves.
	if s.IsRunningInTerminal() && result.ConsumedBy != "" && result.ConsumedBy != consumedByAponoCli {
		return fmt.Errorf("credentials for this session were already used elsewhere. reset them with `apono access reset-credentials %s` and try again", sessionID)
	}

	client, ok := findClient(result.Clients, clientID)
	if !ok {
		return fmt.Errorf("Client %q is not supported yet.\nSupported clients for this session: %s.\nYou can still copy the connection command and run it manually in your preferred client.", clientID, availableIDs(result.Clients))
	}

	combinedCommand := combineSetupAndInvocationCommands(utils.FromNullableString(client.AuthCommand), client.InvocationCommand)

	switch client.LauncherType {
	case ClientKindGUI:
		return s.executeCommand(cobraCmd, combinedCommand)

	case ClientKindTUI:
		if s.IsRunningInTerminal() {
			return s.executeCommand(cobraCmd, combinedCommand)
		}
		// Headless TUI needs a real terminal window; .app/AppleScript invocation has no stdin terminal.
		return s.executeCommand(cobraCmd, s.BuildTerminalLaunchCommand(combinedCommand))

	default:
		return fmt.Errorf("unknown client kind %q for %q", client.LauncherType, clientID)
	}
}

func (s *ClientStarter) executeCommand(cobraCmd *cobra.Command, combined string) error {
	exitCode, stderr, err := s.RunShellCommand(cobraCmd, combined)
	if err != nil {
		return fmt.Errorf("failed to start client: %w\n%s", err, stderr)
	}
	if exitCode != 0 {
		return fmt.Errorf("client exited with code %d\n%s", exitCode, stderr)
	}
	return nil
}

func findClient(clients []clientapi.LauncherClientModel, id string) (clientapi.LauncherClientModel, bool) {
	for _, c := range clients {
		if c.Id == id {
			return c, true
		}
	}
	return clientapi.LauncherClientModel{}, false
}

func availableIDs(clients []clientapi.LauncherClientModel) string {
	ids := make([]string, 0, len(clients))
	for _, c := range clients {
		ids = append(ids, c.Id)
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return "(none)"
	}
	return strings.Join(ids, ", ")
}

func combineSetupAndInvocationCommands(setup, invocation string) string {
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

func runShellCommand(cobraCmd *cobra.Command, command string) (exitCode int, stderrTail string, err error) {
	if strings.TrimSpace(command) == "" {
		return 0, "", fmt.Errorf("empty client command")
	}

	var stderr bytes.Buffer
	cmd := exec.CommandContext(cobraCmd.Context(), "sh", "-c", command)
	cmd.Stdout = cobraCmd.OutOrStdout()
	cmd.Stdin = cobraCmd.InOrStdin()
	cmd.Stderr = &stderr

	err = cmd.Run()
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
func buildTerminalLaunchCommand(command string) string {
	// AppleScript double-quoted string: \ and " need escaping.
	escaped := strings.ReplaceAll(command, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	// Outer wrapping for `sh -c` is single-quoted, so each ' must be closed,
	// emitted literally, and reopened: ' becomes '\''.
	escaped = strings.ReplaceAll(escaped, `'`, `'\''`)
	return fmt.Sprintf(`osascript -e 'tell application "Terminal" to do script "%s"' -e 'tell application "Terminal" to activate'`, escaped)
}

func isRunningInTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
