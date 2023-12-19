package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/apono-io/apono-cli/pkg/clientapi"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

const (
	outputFlagName = "output"
	runFlagName    = "run"

	cliOutputFormat          = "cli"
	linkOutputFormat         = "link"
	instructionsOutputFormat = "instructions"
	jsonOutputFormat         = "json"
)

func AccessDetails() *cobra.Command {
	var outputFormat string
	var shouldExecuteAccessCommand bool

	cmd := &cobra.Command{
		Use:     "use [id]",
		GroupID: Group.ID,
		Short:   "Get access session details",
		Args:    cobra.MinimumNArgs(1), // This will enforce that exactly 1 argument must be provided
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			sessionID := args[0]

			session, _, err := client.ClientAPI.AccessSessionsAPI.GetAccessSession(cmd.Context(), sessionID).Execute()
			if err != nil {
				return fmt.Errorf("access session with id %s not found", sessionID)
			}

			if shouldExecuteAccessCommand {
				return executeAccessDetails(cmd, client, session)
			}

			if !contains(session.ConnectionMethods, outputFormat) {
				return fmt.Errorf("unsupported output format: %s. use one of: %s", outputFormat, strings.Join(session.ConnectionMethods, ", "))
			}

			accessDetails, _, err := client.ClientAPI.AccessSessionsAPI.GetAccessSessionAccessDetails(cmd.Context(), sessionID).Execute()
			if err != nil {
				return err
			}

			var output string
			switch outputFormat {
			case cliOutputFormat:
				output = *accessDetails.Cli.Get()
			case linkOutputFormat:
				link := accessDetails.GetLink()
				output = link.GetUrl()
			case instructionsOutputFormat:
				output = accessDetails.Instructions.Plain
			case jsonOutputFormat:
				var outputBytes []byte
				outputBytes, err = json.Marshal(accessDetails.Json)
				if err != nil {
					return err
				}
				output = string(outputBytes)
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), output)
			if err != nil {
				return err
			}

			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&outputFormat, outputFlagName, "o", "instructions", "output format")
	flags.BoolVarP(&shouldExecuteAccessCommand, runFlagName, "r", false, "execute the cli command")

	return cmd
}

func executeAccessDetails(cobraCmd *cobra.Command, client *aponoapi.AponoClient, session *clientapi.AccessSessionClientModel) error {
	if runtime.GOOS == "windows" {
		return errors.New("--run flag is not supported on windows")
	}

	if !contains(session.ConnectionMethods, cliOutputFormat) {
		return fmt.Errorf("--run flag is not supported for session id %s", session.Id)
	}

	accessDetails, _, err := client.ClientAPI.AccessSessionsAPI.GetAccessSessionAccessDetails(cobraCmd.Context(), session.Id).Execute()
	if err != nil {
		return fmt.Errorf("error getting access details for session id %s: %w", session.Id, err)
	}

	err = executeCommand(cobraCmd, accessDetails.GetCli())
	if err != nil {
		return err
	}

	return nil
}

func executeCommand(cobraCmd *cobra.Command, command string) error {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(cobraCmd.Context(), "sh", "-c", command)
	cmd.Stdout = cobraCmd.OutOrStdout()
	cmd.Stdin = cobraCmd.InOrStdin()
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error executing command:\n%s\n%s", command, stderr.String())
	}

	return nil
}

func contains(availableCommands []string, command string) bool {
	for _, c := range availableCommands {
		if command == c {
			return true
		}
	}
	return false
}
