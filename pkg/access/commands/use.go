package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

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

			accessID := args[0]

			if shouldExecuteAccessCommand {
				return executeAccessDetails(cmd.Context(), client, accessID)
			}

			err = verifyOutputFormatIsSupported(cmd.Context(), client, accessID, outputFormat)
			if err != nil {
				return err
			}

			accessDetails, _, err := client.ClientAPI.AccessSessionsAPI.GetAccessSessionAccessDetails(cmd.Context(), accessID).Execute()
			if err != nil {
				return err
			}

			var output string
			switch outputFormat {
			case cliOutputFormat:
				if accessDetails.GetCli() != "" {
					output = accessDetails.GetCli()
				} else {
					output = accessDetails.Instructions.Plain
				}
			case linkOutputFormat:
				link := accessDetails.GetLink()
				output = link.GetUrl()
			case instructionsOutputFormat:
				output = accessDetails.Instructions.Plain
			case jsonOutputFormat:
				var outputBytes []byte
				outputBytes, err = json.Marshal(accessDetails.Credentials)
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
	flags.StringVarP(&outputFormat, outputFlagName, "o", "cli", "output format")
	flags.BoolVarP(&shouldExecuteAccessCommand, runFlagName, "r", false, "output format")

	return cmd
}

func verifyOutputFormatIsSupported(ctx context.Context, client *aponoapi.AponoClient, id string, outputFormat string) error {
	session, _, err := client.ClientAPI.AccessSessionsAPI.GetAccessSession(ctx, id).Execute()
	if err != nil {
		return fmt.Errorf("access session with id %s not found", id)
	}

	for _, supportedFormat := range session.ConnectionMethods {
		if supportedFormat == outputFormat {
			return nil
		}
	}

	return fmt.Errorf("unsupported output format: %s. use one of: %s", outputFormat, strings.Join(session.ConnectionMethods, ", "))
}

func executeAccessDetails(ctx context.Context, client *aponoapi.AponoClient, accessID string) error {
	if runtime.GOOS == "windows" {
		return errors.New("--run flag is not supported on windows")
	}

	err := verifyOutputFormatIsSupported(ctx, client, accessID, cliOutputFormat)
	if err != nil {
		return fmt.Errorf("--run flag is not supported for session id %s", accessID)
	}

	accessDetails, _, err := client.ClientAPI.AccessSessionsAPI.GetAccessSessionAccessDetails(ctx, accessID).Execute()
	if err != nil {
		return fmt.Errorf("error getting access details for session id %s: %w", accessID, err)
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", accessDetails.GetCli()) //nolint:gosec // This is a command that should be executed for the user
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error executing command:\n%s\n%s", accessDetails.GetCli(), stderr.String())
	}

	return nil
}
