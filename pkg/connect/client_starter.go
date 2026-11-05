package connect

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gookit/color"
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/logshipping"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/styles"
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
	fieldExitCode        = "exit_code"
)

var missingClientMarkers = []string{
	"command not found",
	"no such file or directory",
	"unable to find application",
}

type ClientStarter struct {
	FetchClients               func(context.Context, *aponoapi.AponoClient, string) (*ClientFetchResult, error)
	RunShellCommand            func(*cobra.Command, string) (int, string, error)
	BuildTerminalLaunchCommand func(string) (string, error)
	IsRunningInTerminal        func() bool
	Report                     func(ctx context.Context, level, message string, fields map[string]string)
	ResetCredentials           func(context.Context, *aponoapi.AponoClient, string) error

	secrets  []string
	reissued bool
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
		ResetCredentials: services.ResetSessionCredentials,
	}
}

func (s *ClientStarter) resolveClients(ctx context.Context, apiClient *aponoapi.AponoClient, sessionID, clientID string, isTerminal bool, prefetched *ClientFetchResult) (*ClientFetchResult, error) {
	if prefetched != nil {
		return prefetched, nil
	}
	result, err := s.FetchClients(ctx, apiClient, sessionID)
	if err != nil {
		s.reportLauncher(ctx, logshipping.LevelWarn, "launcher: fetch session details failed", err, sessionID, clientID, "", isTerminal)
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
		s.reportLauncher(ctx, logshipping.LevelWarn, "launcher: credentials already used elsewhere", err, sessionID, clientID, "", isTerminal)
		return err
	}

	client, ok := findClient(result.Clients, clientID)
	if !ok {
		err = fmt.Errorf("client %q is not supported yet.\nSupported clients for this session: %s.\nYou can still copy the connection command and run it manually in your preferred client", clientID, availableIDs(result.Clients))
		s.reportLauncher(ctx, logshipping.LevelWarn, "launcher: client not supported", err, sessionID, clientID, "", isTerminal)
		return err
	}

	launcherType := client.LauncherType
	authCommand := strings.TrimSpace(utils.FromNullableString(client.AuthCommand))
	s.reportLauncher(ctx, logshipping.LevelInfo, "launcher: client resolved", nil, sessionID, clientID, launcherType, isTerminal)

	headlessTerminalLauncher := !isTerminal &&
		(launcherType == ClientKindTUI || launcherType == ClientKindTERMINAL || launcherType == ClientKindCLI)

	if authCommand != "" && !headlessTerminalLauncher {
		exitCode, authErr := s.executeCommand(cobraCmd, authCommand)
		if authErr != nil {
			s.reportCommandFailure(ctx, logshipping.LevelWarn, "launcher: auth command failed", exitCode, nil, sessionID, clientID, launcherType, isTerminal)
			return authErr
		}
	}

	client, err = s.resolveCachedCredentials(cobraCmd, apiClient, sessionID, client, isTerminal, headlessTerminalLauncher)
	if err != nil {
		return err
	}
	invocationCommand := client.InvocationCommand

	s.reportLauncher(ctx, logshipping.LevelInfo, "launcher: launching client", nil, sessionID, clientID, launcherType, isTerminal)

	switch launcherType {
	case ClientKindGUI:
		exitCode, launchErr := s.executeCommand(cobraCmd, invocationCommand)
		if launchErr != nil {
			s.reportCommandFailure(ctx, launchFailureLevel(launchErr), "launcher: GUI launch failed", exitCode, launchErr, sessionID, clientID, launcherType, isTerminal)
			return launchErr
		}
		s.reportLauncher(ctx, logshipping.LevelInfo, "launcher: client launched", nil, sessionID, clientID, launcherType, isTerminal)
		return nil

	case ClientKindTUI, ClientKindTERMINAL, ClientKindCLI:
		if !headlessTerminalLauncher {
			exitCode, launchErr := s.executeCommand(cobraCmd, invocationCommand)
			if launchErr != nil {
				s.reportCommandFailure(ctx, launchFailureLevel(launchErr), "launcher: interactive launch failed", exitCode, launchErr, sessionID, clientID, launcherType, isTerminal)
				return launchErr
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
			s.reportLauncher(ctx, logshipping.LevelWarn, "launcher: build terminal launch command failed", wrapErr, sessionID, clientID, launcherType, isTerminal)
			return fmt.Errorf("build terminal launch command: %w", wrapErr)
		}
		exitCode, launchErr := s.executeCommand(cobraCmd, wrapped)
		if launchErr != nil {
			s.reportCommandFailure(ctx, logshipping.LevelWarn, "launcher: headless launch failed", exitCode, launchErr, sessionID, clientID, launcherType, isTerminal)
			return launchErr
		}
		s.reportLauncher(ctx, logshipping.LevelInfo, "launcher: client launched", nil, sessionID, clientID, launcherType, isTerminal)
		return nil

	default:
		err = fmt.Errorf("unknown client kind %q for %q", launcherType, clientID)
		s.reportLauncher(ctx, logshipping.LevelError, "launcher: unknown launcher kind", err, sessionID, clientID, launcherType, isTerminal)
		return err
	}
}

// resolveCachedCredentials makes sure the cached password the invocation needs exists, and
// substitutes it for the password placeholder. Invocations read the cache either through the
// placeholder or directly from the shell (e.g. `$(base64 -d -i ~/.apono/cache/<session>)`), in
// which case a missing file silently becomes an empty password.
//
// The backend only renders the commands that write the cache while the credentials are new, so a
// missing cache with nothing in this launch to write it means the credentials were already handed
// out (printed instructions, another machine). Only then are the credentials reissued and the
// launcher refreshed; whenever something would write the cache, the launch proceeds as before.
func (s *ClientStarter) resolveCachedCredentials(cobraCmd *cobra.Command, apiClient *aponoapi.AponoClient, sessionID string, client clientapi.LauncherClientModel, isTerminal, headless bool) (clientapi.LauncherClientModel, error) {
	if !usesCachedPassword(client.InvocationCommand, sessionID) {
		return client, nil
	}

	ctx := cobraCmd.Context()
	clientID, launcherType := client.Id, client.LauncherType

	if cachedPasswordMissing(sessionID) && !launchCachesPassword(client, sessionID) {
		if headless {
			// Headless output is discarded, so an error here is invisible: launch as before and let
			// the client report the failed login. A placeholder cannot be substituted, which always
			// failed - keep failing, with an actionable message for the handler log.
			s.reportLauncher(ctx, logshipping.LevelWarn, "launcher: credentials not cached", nil, sessionID, clientID, launcherType, isTerminal)
			if strings.Contains(client.InvocationCommand, passwordPlaceholder) {
				return client, credentialsUnavailableError(sessionID, clientID)
			}
			return client, nil
		}

		refreshed, err := s.reissueCredentials(cobraCmd, apiClient, sessionID, clientID, launcherType, isTerminal)
		if err != nil {
			return client, err
		}
		client = refreshed
	}

	if !strings.Contains(client.InvocationCommand, passwordPlaceholder) {
		return client, nil
	}

	pwd, readErr := readCachedPassword(sessionID)
	if readErr != nil {
		s.reportLauncher(ctx, logshipping.LevelWarn, "launcher: resolve credentials failed", readErr, sessionID, clientID, launcherType, isTerminal)
		return client, fmt.Errorf("resolve credentials: %w", readErr)
	}

	encodedPwd := encodePassword(pwd, client.PasswordEncoding)
	client.InvocationCommand = strings.ReplaceAll(client.InvocationCommand, passwordPlaceholder, encodedPwd)
	s.secrets = []string{pwd, encodedPwd}
	return client, nil
}

func cachedPasswordMissing(sessionID string) bool {
	_, err := os.Stat(cachedPasswordPath(sessionID))
	return errors.Is(err, fs.ErrNotExist)
}

// launchCachesPassword reports whether this launch writes the session's cached password: an auth
// command (rendered only for new credentials; already run, or deferred into the new terminal for
// headless launches) or an invocation that writes the cache itself, like the `cli` launcher.
func launchCachesPassword(client clientapi.LauncherClientModel, sessionID string) bool {
	return strings.TrimSpace(utils.FromNullableString(client.AuthCommand)) != "" ||
		writesCachedPassword(client.InvocationCommand, sessionID)
}

// cacheFileRefPattern matches `.apono/cache/<sessionID>`, but not a longer session id sharing the prefix.
func cacheFileRefPattern(sessionID string) string {
	return regexp.QuoteMeta(cacheDirMarker+sessionID) + `(?:[^A-Za-z0-9._-]|$)`
}

// usesCachedPassword reports whether the command needs the session's cached password, through
// the placeholder or by reading `.apono/cache/<sessionID>` itself.
func usesCachedPassword(command, sessionID string) bool {
	return strings.Contains(command, passwordPlaceholder) ||
		regexp.MustCompile(cacheFileRefPattern(sessionID)).MatchString(command)
}

// writesCachedPassword reports whether the command redirects output into the session's cache file,
// e.g. `base64 > ~/.apono/cache/<sessionID>`.
func writesCachedPassword(command, sessionID string) bool {
	return regexp.MustCompile(`>\s*["']?[^\s"'>]*` + cacheFileRefPattern(sessionID)).MatchString(command)
}

// CredentialsReissued reports whether the last Start reset the session credentials to connect.
func (s *ClientStarter) CredentialsReissued() bool {
	return s.reissued
}

// reissueCredentials resets the session credentials, fetches the launcher again and runs
// its new auth command, which caches the fresh password for the invocation to use.
func (s *ClientStarter) reissueCredentials(cobraCmd *cobra.Command, apiClient *aponoapi.AponoClient, sessionID, clientID, launcherType string, isTerminal bool) (clientapi.LauncherClientModel, error) {
	ctx := cobraCmd.Context()

	_, err := fmt.Fprintf(cobraCmd.OutOrStdout(), "\n%s Credentials for this session are not available on this machine. Resetting credentials to connect with %s...\n", styles.NoticeMsgPrefix, color.Green.Sprint(clientID))
	if err != nil {
		return clientapi.LauncherClientModel{}, err
	}

	err = s.ResetCredentials(ctx, apiClient, sessionID)
	if err != nil {
		s.reportLauncher(ctx, logshipping.LevelWarn, "launcher: reset credentials failed", err, sessionID, clientID, launcherType, isTerminal)
		return clientapi.LauncherClientModel{}, fmt.Errorf("failed to reissue credentials: %w. %w", err, credentialsUnavailableError(sessionID, clientID))
	}

	result, err := s.FetchClients(ctx, apiClient, sessionID)
	if err != nil {
		s.reportLauncher(ctx, logshipping.LevelWarn, "launcher: fetch session details after reset failed", err, sessionID, clientID, launcherType, isTerminal)
		return clientapi.LauncherClientModel{}, fmt.Errorf("could not fetch session details: %w", err)
	}

	client, ok := findClient(result.Clients, clientID)
	if !ok {
		err = fmt.Errorf("client %q is no longer available for session %s", clientID, sessionID)
		s.reportLauncher(ctx, logshipping.LevelWarn, "launcher: client not found after reset", err, sessionID, clientID, launcherType, isTerminal)
		return clientapi.LauncherClientModel{}, err
	}

	if authCommand := strings.TrimSpace(utils.FromNullableString(client.AuthCommand)); authCommand != "" {
		exitCode, authErr := s.executeCommand(cobraCmd, authCommand)
		if authErr != nil {
			s.reportCommandFailure(ctx, logshipping.LevelWarn, "launcher: auth command after reset failed", exitCode, nil, sessionID, clientID, launcherType, isTerminal)
			return clientapi.LauncherClientModel{}, authErr
		}
	}

	s.reissued = true
	_, err = fmt.Fprintf(cobraCmd.OutOrStdout(), "%s Credentials reset successfully, connecting...\n\n", styles.NoticeMsgPrefix)
	if err != nil {
		return clientapi.LauncherClientModel{}, err
	}
	s.reportLauncher(ctx, logshipping.LevelInfo, "launcher: credentials reissued", nil, sessionID, clientID, launcherType, isTerminal)
	return client, nil
}

func credentialsUnavailableError(sessionID, clientID string) error {
	return fmt.Errorf("credentials for this session were already viewed and are not available on this machine. to connect with %s, reset them with `apono access reset-credentials %s` and try again", clientID, sessionID)
}

func (s *ClientStarter) executeCommand(cobraCmd *cobra.Command, command string) (int, error) {
	exitCode, stderr, err := s.RunShellCommand(cobraCmd, command)
	if err != nil {
		return exitCode, fmt.Errorf("failed to start client: %w\n%s", err, stderr)
	}
	if exitCode != 0 {
		return exitCode, fmt.Errorf("client exited with code %d\n%s", exitCode, stderr)
	}
	return exitCode, nil
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
	fields := launcherFields(sessionID, clientID, launcherType, isTerminal)
	if cause != nil {
		fields[fieldError] = withoutSecrets(cause, s.secrets).Error()
	}
	s.Report(ctx, level, message, fields)
}

func (s *ClientStarter) reportCommandFailure(ctx context.Context, level, message string, exitCode int, cause error, sessionID, clientID, launcherType string, isTerminal bool) {
	if s.Report == nil {
		return
	}
	fields := launcherFields(sessionID, clientID, launcherType, isTerminal)
	fields[fieldExitCode] = strconv.Itoa(exitCode)
	if cause != nil {
		fields[fieldError] = withoutSecrets(cause, s.secrets).Error()
	}
	s.Report(ctx, level, message, fields)
}

func launchFailureLevel(cause error) string {
	if cause == nil {
		return logshipping.LevelError
	}
	message := strings.ToLower(cause.Error())
	for _, marker := range missingClientMarkers {
		if strings.Contains(message, marker) {
			return logshipping.LevelWarn
		}
	}
	return logshipping.LevelError
}

func launcherFields(sessionID, clientID, launcherType string, isTerminal bool) map[string]string {
	return map[string]string{
		fieldAccessSessionID: sessionID,
		fieldClientID:        clientID,
		fieldLauncherType:    launcherType,
		fieldIsTerminal:      strconv.FormatBool(isTerminal),
	}
}
