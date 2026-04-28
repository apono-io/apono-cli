package actions

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/interactive/flows"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/utils"

	"github.com/spf13/cobra"
)

const (
	outputFlagName = "output"
	runFlagName    = "run"
	clientFlagName = "client"
)

type accessUseCommandFlags struct {
	outputFormat               string
	shouldExecuteAccessCommand bool
	launcherID                 string
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
				if cmdFlags.launcherID == "" {
					return fmt.Errorf("--client requires a launcher name (e.g. --client dbeaver)")
				}
				if runtime.GOOS != "darwin" {
					return fmt.Errorf("--client is only supported on macOS; use --run to launch in your current terminal")
				}
				// Pre-GA gate: Kinda instead of FF mechanims
				if os.Getenv("APONO_LAUNCHER_PREVIEW") != "1" {
					return fmt.Errorf("--client is in preview and not yet available; use --run to launch in your current terminal")
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
				return services.NewLauncher().LaunchSession(cmd, client, session.Id, cmdFlags.launcherID)
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
	flags.StringVar(&cmdFlags.launcherID, clientFlagName, "", "launch the session in the named client (e.g. dbeaver, tableplus, k9s, cli) — macOS only")

	cmd.MarkFlagsMutuallyExclusive(runFlagName, clientFlagName)

	return cmd
}

func resolveOutputFormat(cmdFlags *accessUseCommandFlags) string {
	if cmdFlags.shouldExecuteAccessCommand {
		return services.CliOutputFormat
	}

	return cmdFlags.outputFormat
}
