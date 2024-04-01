package actions

import (
	"fmt"
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
)

type accessUseCommandFlags struct {
	outputFormat               string
	shouldExecuteAccessCommand bool
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

			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			session, _, err := client.ClientAPI.AccessSessionsAPI.GetAccessSession(cmd.Context(), args[0]).Execute()
			if err != nil {
				return fmt.Errorf("access session with id %s not found", args[0])
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

	return cmd
}

func resolveOutputFormat(cmdFlags *accessUseCommandFlags) string {
	if cmdFlags.shouldExecuteAccessCommand {
		return services.CliOutputFormat
	}

	return cmdFlags.outputFormat
}
