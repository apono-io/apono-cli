package connect

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/logshipping"
	"github.com/apono-io/apono-cli/pkg/terminal"
	"github.com/apono-io/apono-cli/pkg/utils"
)

const (
	ClientKindGUI      = "GUI"
	ClientKindTUI      = "TUI"
	ClientKindTERMINAL = "TERMINAL"
	ClientKindCLI      = "CLI"
)

const (
	fieldClientID     = "client_id"
	fieldLauncherType = "launcher_type"
	fieldIsTerminal   = "is_terminal"
)

type ClientStarter struct {
	FetchClients               func(context.Context, *aponoapi.AponoClient, string) (*ClientFetchResult, error)
	RunShellCommand            func(*cobra.Command, string) (int, string, error)
	BuildTerminalLaunchCommand func(string) (string, error)
	IsRunningInTerminal        func() bool
}

func NewClientStarter() *ClientStarter {
	return &ClientStarter{
		FetchClients:               FetchClients,
		RunShellCommand:            runShellCommand,
		BuildTerminalLaunchCommand: terminal.BuildLaunchCommand,
		IsRunningInTerminal:        isRunningInTerminal,
	}
}

func (s *ClientStarter) Start(cobraCmd *cobra.Command, apiClient *aponoapi.AponoClient, sessionID, clientID string) error {
	ctx := cobraCmd.Context()
	isTerminal := s.IsRunningInTerminal()

	result, err := s.FetchClients(ctx, apiClient, sessionID)
	if err != nil {
		reportLauncherError(ctx, "launcher: fetch session details failed", sessionID, clientID, "", isTerminal)
		return fmt.Errorf("could not fetch session details: %w", err)
	}

	// Portal and Slack show their own "credentials already in use" prompt before
	// firing the apono:// URI, so a headless (executed from protocol handler) run can trust that. A terminal user typed
	// the command directly and never saw that prompt - surface it here ourselves.
	if isTerminal && result.ConsumedBy != "" && result.ConsumedBy != aponoapi.ConsumedByAponoCli {
		reportLauncherError(ctx, "launcher: credentials already used elsewhere", sessionID, clientID, "", isTerminal)
		return fmt.Errorf("credentials for this session were already used elsewhere. reset them with `apono access reset-credentials %s` and try again", sessionID)
	}

	client, ok := findClient(result.Clients, clientID)
	if !ok {
		reportLauncherError(ctx, "launcher: client not supported", sessionID, clientID, "", isTerminal)
		return fmt.Errorf("client %q is not supported yet.\nSupported clients for this session: %s.\nYou can still copy the connection command and run it manually in your preferred client", clientID, availableIDs(result.Clients))
	}

	launcherType := client.LauncherType
	authCommand := strings.TrimSpace(utils.FromNullableString(client.AuthCommand))
	invocationCommand := client.InvocationCommand

	headlessTerminalLauncher := !isTerminal &&
		(launcherType == ClientKindTUI || launcherType == ClientKindTERMINAL || launcherType == ClientKindCLI)

	if authCommand != "" && !headlessTerminalLauncher {
		if err := s.executeCommand(cobraCmd, authCommand); err != nil {
			reportLauncherError(ctx, "launcher: auth command failed", sessionID, clientID, launcherType, isTerminal)
			return err
		}
	}

	if strings.Contains(invocationCommand, passwordPlaceholder) {
		pwd, err := readCachedPassword(sessionID)
		if err != nil {
			reportLauncherError(ctx, "launcher: resolve credentials failed", sessionID, clientID, launcherType, isTerminal)
			return fmt.Errorf("resolve credentials: %w", err)
		}
		invocationCommand = strings.ReplaceAll(invocationCommand, passwordPlaceholder, encodePassword(pwd, client.PasswordEncoding))
	}

	switch launcherType {
	case ClientKindGUI:
		if err := s.executeCommand(cobraCmd, invocationCommand); err != nil {
			reportLauncherError(ctx, "launcher: GUI launch failed", sessionID, clientID, launcherType, isTerminal)
			return err
		}
		return nil

	case ClientKindTUI, ClientKindTERMINAL, ClientKindCLI:
		if !headlessTerminalLauncher {
			if err := s.executeCommand(cobraCmd, invocationCommand); err != nil {
				reportLauncherError(ctx, "launcher: interactive launch failed", sessionID, clientID, launcherType, isTerminal)
				return err
			}
			return nil
		}
		combined := invocationCommand
		if authCommand != "" {
			combined = authCommand + " && " + invocationCommand
		}
		wrapped, err := s.BuildTerminalLaunchCommand(combined)
		if err != nil {
			reportLauncherError(ctx, "launcher: build terminal launch command failed", sessionID, clientID, launcherType, isTerminal)
			return fmt.Errorf("build terminal launch command: %w", err)
		}
		if err := s.executeCommand(cobraCmd, wrapped); err != nil {
			reportLauncherError(ctx, "launcher: headless launch failed", sessionID, clientID, launcherType, isTerminal)
			return err
		}
		return nil

	default:
		reportLauncherError(ctx, "launcher: unknown launcher kind", sessionID, clientID, launcherType, isTerminal)
		return fmt.Errorf("unknown client kind %q for %q", launcherType, clientID)
	}
}

func (s *ClientStarter) executeCommand(cobraCmd *cobra.Command, command string) error {
	exitCode, stderr, err := s.RunShellCommand(cobraCmd, command)
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

func isRunningInTerminal() bool {
	return terminal.IsRunning(os.Stdin)
}

func reportLauncherError(ctx context.Context, message, sessionID, clientID, launcherType string, isTerminal bool) {
	logshipping.Report(ctx, sessionID, logshipping.LevelError, message, map[string]string{
		fieldClientID:     clientID,
		fieldLauncherType: launcherType,
		fieldIsTerminal:   strconv.FormatBool(isTerminal),
	})
}
