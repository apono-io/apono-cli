package actions

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/analytics"
	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/config"
	"github.com/apono-io/apono-cli/pkg/connect"
	"github.com/apono-io/apono-cli/pkg/interactive/flows"
	"github.com/apono-io/apono-cli/pkg/logshipping"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/utils"
)

const (
	outputFlagName  = "output"
	runFlagName     = "run"
	clientFlagName  = "client"
	profileFlagName = "profile"

	accountIDEnvVar = "_APONO_ACCOUNT_ID_"
)

const sessionURLEncodedEnvVar = "_APONO_SESSION_URL_ENCODED_"

const (
	launcherLogPrefix  = "launcher:"
	accessUseLogPrefix = "access use:"

	fieldAccessSessionID = "access_session_id"
	fieldClientID        = "client_id"
)

type accessUseCommandFlags struct {
	outputFormat               string
	shouldExecuteAccessCommand bool
	clientID                   string
}

func AccessDetails() *cobra.Command {
	cmdFlags := &accessUseCommandFlags{}

	cmd := &cobra.Command{
		Use:   "use <session_id>",
		Short: "Get access session details",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("missing session id")
			}

			if cmd.Flags().Changed(clientFlagName) {
				if cmdFlags.clientID == "" {
					return fmt.Errorf("--client requires a client name (e.g. --client dbeaver)")
				}
				if runtime.GOOS != "darwin" {
					return fmt.Errorf("--client is only supported on macOS; use --run to launch in your current terminal")
				}
			}
			if accountID := os.Getenv(accountIDEnvVar); accountID != "" {
				if cmd.Flags().Changed(profileFlagName) {
					return fmt.Errorf("%s and --profile are mutually exclusive", accountIDEnvVar)
				}
				if err := overrideProfileClientByAccountID(cmd, accountID); err != nil {
					return err
				}
			}

			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			shipHandlerLog(cmd.Context())

			sessionURLEncoded := os.Getenv(sessionURLEncodedEnvVar) != ""
			_ = os.Unsetenv(sessionURLEncodedEnvVar)
			sessionID, err := resolveSessionIDArg(args[0], sessionURLEncoded)
			if err != nil {
				return err
			}

			session, err := connect.FetchSession(cmd.Context(), client, sessionID)
			if err != nil {
				reportSessionFetchFailure(cmd.Context(), sessionID, cmdFlags.clientID, err)
				return err
			}

			if cmd.Flags().Changed(clientFlagName) {
				err = connect.NewClientStarter().Start(cmd, client, session.Id, cmdFlags.clientID, nil)
				if err != nil {
					return err
				}
				launchedFromBrowser := os.Getenv(accountIDEnvVar) != ""
				origin := analytics.OriginFlagRun
				if launchedFromBrowser {
					origin = analytics.OriginBrowser
				}
				analytics.SendLaunchClientEvent(cmd.Context(), cmdFlags.clientID, session.Id, session.Integration.Type, origin)
				return nil
			}

			if len(session.ConnectionMethods) == 0 {
				return fmt.Errorf("no available connection methods")
			}

			connectionDetailsOutputFormat := resolveOutputFormat(cmdFlags)

			if cmdFlags.shouldExecuteAccessCommand && connectionDetailsOutputFormat == services.CliOutputFormat {
				return executeAccessCommand(cmd, client, session)
			}

			if !utils.Contains(session.ConnectionMethods, connectionDetailsOutputFormat) {
				return fmt.Errorf("unsupported output format: %s. use one of: %s", connectionDetailsOutputFormat, strings.Join(session.ConnectionMethods, ", "))
			}

			accessDetails, customInstructionMessage, err := services.GetSessionDetails(cmd.Context(), client, session.Id, connectionDetailsOutputFormat)
			if err != nil {
				reportAccessDetailsFetchFailure(cmd.Context(), session.Id, err)
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), accessDetails)
			if err != nil {
				return err
			}

			if customInstructionMessage != "" {
				err = services.PrintCustomInstructionMessage(cmd, customInstructionMessage)
				if err != nil {
					return err
				}
			}

			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&cmdFlags.outputFormat, outputFlagName, "o", "instructions", fmt.Sprintf("output format: %s | %s | %s | %s", services.CliOutputFormat, services.LinkOutputFormat, services.InstructionsOutputFormat, services.JSONOutputFormat))
	flags.BoolVarP(&cmdFlags.shouldExecuteAccessCommand, runFlagName, "r", false, "execute the cli command")
	flags.StringVarP(&cmdFlags.clientID, clientFlagName, "c", "", "Launch the session in a supported local `client`. Supported on macOS only.\nSupported clients: dbeaver, tableplus, k9s")

	cmd.MarkFlagsMutuallyExclusive(runFlagName, clientFlagName)

	return cmd
}

func executeAccessCommand(cmd *cobra.Command, client *aponoapi.AponoClient, session *clientapi.AccessSessionClientModel) error {
	err := services.ExecuteAccessDetails(cmd, client, session)
	reportAccessDetailsFetchFailure(cmd.Context(), session.Id, err)
	if isAccessSessionUnavailable(err) {
		return err
	}
	return flows.PrintErrorConnectingSuggestionOnFailure(cmd, session, err)
}

func reportSessionFetchFailure(ctx context.Context, sessionID, clientID string, err error) {
	logPrefix := accessUseLogPrefix
	if clientID != "" {
		logPrefix = launcherLogPrefix
	}

	var unavailableErr *aponoapi.AccessSessionUnavailableError
	switch {
	case errors.Is(err, context.Canceled):
		return
	case errors.As(err, &unavailableErr) && unavailableErr.Reason == aponoapi.AccessSessionReasonNotFound:
		reportFetchFailure(ctx, logshipping.LevelWarn, logPrefix+" session not found", sessionID, clientID, err)
	case errors.As(err, &unavailableErr):
		reportFetchFailure(ctx, logshipping.LevelWarn, logPrefix+" session not active", sessionID, clientID, err)
	default:
		reportFetchFailure(ctx, logshipping.LevelError, logPrefix+" fetch session failed", sessionID, clientID, err)
	}
}

func reportAccessDetailsFetchFailure(ctx context.Context, sessionID string, err error) {
	message := accessUseLogPrefix + " fetch session details failed"

	var apiErr *aponoapi.APIError
	switch {
	case isAccessSessionUnavailable(err):
		reportFetchFailure(ctx, logshipping.LevelWarn, message, sessionID, "", err)
	case errors.As(err, &apiErr):
		reportFetchFailure(ctx, logshipping.LevelError, message, sessionID, "", err)
	}
}

func reportFetchFailure(ctx context.Context, level, message, sessionID, clientID string, err error) {
	httpStatus, reason := aponoapi.FailureFields(err)
	logshipping.Report(ctx, logshipping.CallerCLI, level, message, map[string]string{
		fieldAccessSessionID:            sessionID,
		fieldClientID:                   clientID,
		aponoapi.FailureFieldHTTPStatus: httpStatus,
		aponoapi.FailureFieldReason:     reason,
	})
}

func isAccessSessionUnavailable(err error) bool {
	var unavailableErr *aponoapi.AccessSessionUnavailableError
	return errors.As(err, &unavailableErr)
}

func overrideProfileClientByAccountID(cmd *cobra.Command, accountID string) error {
	profileName, err := config.GetProfileByAccountID(accountID)
	if err != nil {
		return fmt.Errorf("not logged in to account %s. Run `apono login` to add it", accountID)
	}

	overrideClient, err := aponoapi.CreateClient(cmd.Context(), string(profileName))
	if err != nil {
		return fmt.Errorf("failed to authenticate to account %s: %w", accountID, err)
	}

	cmd.SetContext(aponoapi.CreateClientContext(cmd.Context(), overrideClient))
	cmd.SetContext(config.CreateProfileContext(cmd.Context(), string(profileName)))
	return nil
}

func resolveSessionIDArg(arg string, urlEncoded bool) (string, error) {
	if !urlEncoded {
		return arg, nil
	}
	decoded, err := url.PathUnescape(arg)
	if err != nil {
		return "", fmt.Errorf("invalid session id in launch URL: %w", err)
	}
	for _, r := range decoded {
		if r < 0x20 || r == 0x7f {
			return "", fmt.Errorf("invalid session id in launch URL: contains control characters")
		}
	}
	return decoded, nil
}

func resolveOutputFormat(cmdFlags *accessUseCommandFlags) string {
	if cmdFlags.shouldExecuteAccessCommand {
		return services.CliOutputFormat
	}

	return cmdFlags.outputFormat
}
