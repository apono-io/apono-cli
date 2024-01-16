package actions

import (
	"fmt"
	"strings"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/services"

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
				return services.ExecuteAccessDetails(cmd, client, session)
			}

			if !contains(session.ConnectionMethods, connectionDetailsOutputFormat) {
				return fmt.Errorf("unsupported output format: %s. use one of: %s", connectionDetailsOutputFormat, strings.Join(session.ConnectionMethods, ", "))
			}

			accessDetails, err := services.GetSessionDetails(cmd.Context(), client, session.Id, connectionDetailsOutputFormat)
			if err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), accessDetails)

			return err
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&cmdFlags.outputFormat, outputFlagName, "o", "instructions", fmt.Sprintf("output format: %s | %s | %s | %s", services.CliOutputFormat, services.LinkOutputFormat, services.InstructionsOutputFormat, services.JSONOutputFormat))
	flags.BoolVarP(&cmdFlags.shouldExecuteAccessCommand, runFlagName, "r", false, "execute the cli command")

	return cmd
}

func contains(availableCommands []string, command string) bool {
	for _, c := range availableCommands {
		if command == c {
			return true
		}
	}
	return false
}

func resolveOutputFormat(cmdFlags *accessUseCommandFlags) string {
	if cmdFlags.shouldExecuteAccessCommand {
		return services.CliOutputFormat
	}

	return cmdFlags.outputFormat
}
