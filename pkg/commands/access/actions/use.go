package actions

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/config"
	"github.com/apono-io/apono-cli/pkg/connect"
	"github.com/apono-io/apono-cli/pkg/interactive/flows"
	"github.com/apono-io/apono-cli/pkg/interactive/selectors"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/utils"
)

const (
	outputFlagName  = "output"
	runFlagName     = "run"
	clientFlagName  = "client"
	launchFlagName  = "launch"
	profileFlagName = "profile"

	accountIDEnvVar = "_APONO_ACCOUNT_ID_"
)

type accessUseCommandFlags struct {
	outputFormat               string
	shouldExecuteAccessCommand bool
	clientID                   string
	shouldLaunchInteractive    bool
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
				if runtime.GOOS != utils.DarwinOS {
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

			session, _, err := client.ClientAPI.AccessSessionsAPI.GetAccessSession(cmd.Context(), args[0]).Execute()
			if err != nil {
				return fmt.Errorf("access session with id %s not found", args[0])
			}

			if cmdFlags.shouldLaunchInteractive {
				return runLaunchInteractive(cmd, client, session)
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
	flags.StringVarP(&cmdFlags.clientID, clientFlagName, "c", "", "Launch the session in a supported local `client`. Supported on macOS only.\nSupported clients: dbeaver, tableplus, k9s")
	flags.BoolVarP(&cmdFlags.shouldLaunchInteractive, launchFlagName, "l", false, "Pick a local client interactively from those installed. Supported on macOS only.")

	cmd.MarkFlagsMutuallyExclusive(runFlagName, clientFlagName, launchFlagName)

	return cmd
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

func resolveOutputFormat(cmdFlags *accessUseCommandFlags) string {
	if cmdFlags.shouldExecuteAccessCommand {
		return services.CliOutputFormat
	}

	return cmdFlags.outputFormat
}

func runLaunchInteractive(cmd *cobra.Command, client *aponoapi.AponoClient, session *clientapi.AccessSessionClientModel) error {
	if runtime.GOOS != utils.DarwinOS {
		return fmt.Errorf("--launch is only supported on macOS")
	}
	result, err := connect.FetchClients(cmd.Context(), client, session.Id)
	if err != nil {
		return fmt.Errorf("could not fetch session details: %w", err)
	}
	if len(result.Clients) == 0 {
		return fmt.Errorf("no supported launchers for this session")
	}
	var installed []clientapi.LauncherClientModel
	for _, c := range result.Clients {
		if connect.IsInstalled(c) {
			installed = append(installed, c)
		}
	}
	if len(installed) == 0 {
		return noInstalledClientsError(result.Clients)
	}
	if err := flows.PrintErrorConnectingSuggestion(cmd, session.Id); err != nil {
		return err
	}
	selectedID, err := selectors.RunLauncherClientSelector(installed)
	if err != nil {
		return err
	}
	return connect.NewClientStarter().Start(cmd, client, session.Id, selectedID)
}

func noInstalledClientsError(clients []clientapi.LauncherClientModel) error {
	var guiClientNames []string
	for _, c := range clients {
		if c.LauncherType == connect.ClientKindGUI || c.LauncherType == connect.ClientKindTUI {
			guiClientNames = append(guiClientNames, c.DisplayName)
		}
	}
	if len(guiClientNames) == 0 {
		return fmt.Errorf("no GUI or TUI launchers are configured for this session. Use --run to execute the inline command in your terminal")
	}
	return fmt.Errorf("no installed clients found. Install a CLI tool the session uses, or one of: %s", strings.Join(guiClientNames, ", "))
}
