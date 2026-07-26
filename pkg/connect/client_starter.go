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
	fieldAccessSessionID = "access_session_id"
	fieldClientID        = "client_id"
	fieldLauncherType    = "launcher_type"
	fieldIsTerminal      = "is_terminal"
	fieldError           = "error"
)

type ClientStarter struct {
	FetchClients               func(context.Context, *aponoapi.AponoClient, string) (*ClientFetchResult, error)
	RunShellCommand            func(*cobra.Command, string) (int, string, error)
	BuildTerminalLaunchCommand func(string) (string, error)
	IsRunningInTerminal        func() bool
	Report                     func(ctx context.Context, level, message string, fields map[string]string)
}

func NewClientStarter() *ClientStarter {
	return &ClientStarter{
		FetchClients:               FetchClients,
		RunShellCommand:            runShellCommand,
		BuildTerminalLaunchCommand: terminal.BuildLaunchCommand,
		IsRunningInTerminal:        isRunningInTerminal,
		Report: func(ctx context.Context, level, message string, fields map[string]string) {
			logshipping.Report(ctx, logshipping.CallerCLI, level, message, fields)
		},
	}
}

func (s *ClientStarter) resolveClients(ctx context.Context, apiClient *aponoapi.AponoClient, sessionID, clientID string, isTerminal bool, prefetched *ClientFetchResult) (*ClientFetchResult, error) {
	if prefetched != nil {
		return prefetched, nil
	}
	result, err := s.FetchClients(ctx, apiClient, sessionID)
	if err != nil {
		s.reportLauncher(ctx, logshipping.LevelError, "launcher: fetch session details failed", err, sessionID, clientID, "", isTerminal)
		return nil, fmt.Errorf("could not fetch session details: %w", err)
	}
	s.reportLauncher(ctx, logshipping.LevelInfo, "launcher: session details fetched", nil, sessionID, clientID, "", isTerminal)
	return result, nil
}

func (s *ClientStarter) Start(cobraCmd *cobra.Command, apiClient *aponoapi.AponoClient, sessionID, clientID string, prefetched *ClientFetchResult) error {
	ctx := cobraCmd.Context()
	isTerminal := s.IsRunningInTerminal()

	s.reportLauncher(ctx, logshipping.LevelInfo, "launcher: starting", nil, sessionID, clientID, "", isTerminal)

	result, err := s.resolveClients(ctx, apiClient, sessionID, clientID, isTerminal, prefetched)
	if err != nil {
		return err
	}

	// Portal and Slack show their own "credentials already in use" prompt before
	// firing the apono:// URI, so a headless (executed from protocol handler) run can trust that. A terminal user typed
	// the command directly and never saw that prompt - surface it here ourselves.
	if isTerminal && result.ConsumedBy != "" && result.ConsumedBy != aponoapi.ConsumedByAponoCli {
		err = fmt.Errorf("credentials for this session were already used elsewhere. reset them with `apono access reset-credentials %s` and try again", sessionID)
		s.reportLauncher(ctx, logshipping.LevelError, "launcher: credentials already used elsewhere", err, sessionID, clientID, "", isTerminal)
		return err
	}

	client, ok := findClient(result.Clients, clientID)
	if !ok {
		err = fmt.Errorf("client %q is not supported yet.\nSupported clients for this session: %s.\nYou can still copy the connection command and run it manually in your preferred client", clientID, availableIDs(result.Clients))
		s.reportLauncher(ctx, logshipping.LevelError, "launcher: client not supported", err, sessionID, clientID, "", isTerminal)
		return err
	}

	launcherType := client.LauncherType
	authCommand := strings.TrimSpace(utils.FromNullableString(client.AuthCommand))
	invocationCommand := client.InvocationCommand
	s.reportLauncher(ctx, logshipping.LevelInfo, "launcher: client resolved", nil, sessionID, clientID, launcherType, isTerminal)

	headlessTerminalLauncher := !isTerminal &&
		(launcherType == ClientKindTUI || launcherType == ClientKindTERMINAL || launcherType == ClientKindCLI)

	if authCommand != "" && !headlessTerminalLauncher {
		if err = s.executeCommand(cobraCmd, authCommand); err != nil {
			s.reportLauncher(ctx, logshipping.LevelError, "launcher: auth command failed", err, sessionID, clientID, launcherType, isTerminal)
			return err
		}
	}

	if strings.Contains(invocationCommand, passwordPlaceholder) {
		pwd, readErr := readCachedPassword(sessionID)
		if readErr != nil {
			s.reportLauncher(ctx, logshipping.LevelError, "launcher: resolve credentials failed", readErr, sessionID, clientID, launcherType, isTerminal)
			return fmt.Errorf("resolve credentials: %w", readErr)
		}
		invocationCommand = strings.ReplaceAll(invocationCommand, passwordPlaceholder, encodePassword(pwd, client.PasswordEncoding))
	}

	s.reportLauncher(ctx, logshipping.LevelInfo, "launcher: launching client", nil, sessionID, clientID, launcherType, isTerminal)

	switch launcherType {
	case ClientKindGUI:
		if err = s.executeCommand(cobraCmd, invocationCommand); err != nil {
			s.reportLauncher(ctx, logshipping.LevelError, "launcher: GUI launch failed", err, sessionID, clientID, launcherType, isTerminal)
			return err
		}
		s.reportLauncher(ctx, logshipping.LevelInfo, "launcher: client launched", nil, sessionID, clientID, launcherType, isTerminal)
		return nil

	case ClientKindTUI, ClientKindTERMINAL, ClientKindCLI:
		if !headlessTerminalLauncher {
			if err = s.executeCommand(cobraCmd, invocationCommand); err != nil {
				s.reportLauncher(ctx, logshipping.LevelError, "launcher: interactive launch failed", err, sessionID, clientID, launcherType, isTerminal)
				return err
			}
			s.reportLauncher(ctx, logshipping.LevelInfo, "launcher: client launched", nil, sessionID, clientID, launcherType, isTerminal)
			return nil
		}
		combined := invocationCommand
		if authCommand != "" {
			combined = authCommand + " && " + invocationCommand
		}
		wrapped, wrapErr := s.BuildTerminalLaunchCommand(combined)
		if wrapErr != nil {
			s.reportLauncher(ctx, logshipping.LevelError, "launcher: build terminal launch command failed", wrapErr, sessionID, clientID, launcherType, isTerminal)
			return fmt.Errorf("build terminal launch command: %w", wrapErr)
		}
		if err = s.executeCommand(cobraCmd, wrapped); err != nil {
			s.reportLauncher(ctx, logshipping.LevelError, "launcher: headless launch failed", err, sessionID, clientID, launcherType, isTerminal)
			return err
		}
		s.reportLauncher(ctx, logshipping.LevelInfo, "launcher: client launched", nil, sessionID, clientID, launcherType, isTerminal)
		return nil

	default:
		err = fmt.Errorf("unknown client kind %q for %q", launcherType, clientID)
		s.reportLauncher(ctx, logshipping.LevelError, "launcher: unknown launcher kind", err, sessionID, clientID, launcherType, isTerminal)
		return err
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

func (s *ClientStarter) reportLauncher(ctx context.Context, level, message string, cause error, sessionID, clientID, launcherType string, isTerminal bool) {
	if s.Report == nil {
		return
	}
	fields := map[string]string{
		fieldAccessSessionID: sessionID,
		fieldClientID:        clientID,
		fieldLauncherType:    launcherType,
		fieldIsTerminal:      strconv.FormatBool(isTerminal),
	}
	if cause != nil {
		fields[fieldError] = cause.Error()
	}
	s.Report(ctx, level, message, fields)
}
