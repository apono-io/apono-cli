package actions

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/config"
	"github.com/apono-io/apono-cli/pkg/connect"
	"github.com/apono-io/apono-cli/pkg/interactive/flows"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/utils"
)

const (
	outputFlagName  = "output"
	runFlagName     = "run"
	clientFlagName  = "client"
	accountFlagName = "account"
	profileFlagName = "profile"
)

type accessUseCommandFlags struct {
	outputFormat               string
	shouldExecuteAccessCommand bool
	clientID                   string
	accountID                  string
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

			if cmd.Flags().Changed(accountFlagName) {
				if cmdFlags.accountID == "" {
					return fmt.Errorf("--account requires an account ID")
				}
				if cmd.Flags().Changed(profileFlagName) {
					return fmt.Errorf("--account and --profile are mutually exclusive")
				}
				if err := overrideClientByAccountID(cmd, cmdFlags.accountID); err != nil {
					return err
				}
			}

			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			session, _, err := client.ClientAPI.AccessSessionsAPI.GetAccessSession(cmd.Context(), args[0]).Execute()
			if err != nil {
				return fmt.Errorf("access session with id %s not found", args[0])
			}

			if cmd.Flags().Changed(clientFlagName) {
				return connect.NewClientStarter().Start(cmd, client, session.Id, cmdFlags.clientID)
			}

			if len(session.ConnectionMethods) == 0 {
				return fmt.Errorf("no available connection methods")
			}

			connectionDetailsOutputFormat := resolveOutputFormat(cmdFlags)

			if cmdFlags.shouldExecuteAccessCommand && connectionDetailsOutputFormat == services.CliOutputFormat {
				err = flows.PrintErrorConnectingSuggestion(cmd, session.Id)
				if err != nil {
					return err
				}

				return services.ExecuteAccessDetails(cmd, client, session)
			}

			if !utils.Contains(session.ConnectionMethods, connectionDetailsOutputFormat) {
				return fmt.Errorf("unsupported output format: %s. use one of: %s", connectionDetailsOutputFormat, strings.Join(session.ConnectionMethods, ", "))
			}

			accessDetails, customInstructionMessage, err := services.GetSessionDetails(cmd.Context(), client, session.Id, connectionDetailsOutputFormat)
			if err != nil {
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
	flags.StringVar(&cmdFlags.clientID, clientFlagName, "", "start the named client app for this session (e.g. dbeaver, tableplus, k9s, cli) — macOS only")
	flags.StringVar(&cmdFlags.accountID, accountFlagName, "", "use the profile that belongs to this Apono account ID instead of the active profile (used by the apono:// protocol handler)")

	cmd.MarkFlagsMutuallyExclusive(runFlagName, clientFlagName)

	return cmd
}

// overrideClientByAccountID swaps the context's client to one built from the
// profile matching accountID. The active profile on disk is untouched.
func overrideClientByAccountID(cmd *cobra.Command, accountID string) error {
	profileName, _, err := config.GetProfileByAccountID(accountID)
	if err != nil {
		return connect.SurfaceError(fmt.Errorf("not logged in to account %s. Run `apono login` to add it", accountID))
	}

	overrideClient, err := aponoapi.CreateClient(cmd.Context(), string(profileName))
	if err != nil {
		return connect.SurfaceError(fmt.Errorf("failed to authenticate to account %s: %w", accountID, err))
	}

	cmd.SetContext(aponoapi.CreateClientContext(cmd.Context(), overrideClient))
	cmd.SetContext(config.CreateProfileContext(cmd.Context(), string(profileName)))
	return nil
}

func resolveOutputFormat(cmdFlags *accessUseCommandFlags) string {
	if cmdFlags.shouldExecuteAccessCommand {
		return services.CliOutputFormat
	}

	return cmdFlags.outputFormat
}
