package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
)

const (
	outputFlagName  = "output"
	executeFlagName = "run"
)

func AccessDetails() *cobra.Command {
	var outputFormat string
	var execute bool

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

			accessDetails, _, err := client.ClientAPI.AccessSessionsAPI.GetAccessSessionAccessDetails(cmd.Context(), accessID).Execute()
			if err != nil {
				return err
			}

			if execute {
				return executeAccessDetails(cmd.Context(), accessDetails)
			}

			err = verifyOutputFormatIsSupported(cmd.Context(), client, accessID, outputFormat)
			if err != nil {
				return err
			}

			var output string
			switch outputFormat {
			case "cli":
				if accessDetails.GetCli() != "" {
					output = accessDetails.GetCli()
				} else {
					output = accessDetails.Instructions.Plain
				}
			case "link":
				link := accessDetails.GetLink()
				output = link.GetUrl()
			case "instructions":
				output = accessDetails.Instructions.Plain
			case "json":
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
	flags.BoolVarP(&execute, executeFlagName, "r", false, "output format")

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

func executeAccessDetails(ctx context.Context, accessDetails *clientapi.AccessSessionDetailsClientModel) error {
	if accessDetails.GetCli() == "" {
		return errors.New("access details does not support cli execution")
	}

	err := exec.CommandContext(ctx, "sh", "-c", accessDetails.GetCli()).Run() //nolint:gosec // This is a command that should be executed for the user
	if err != nil {
		return fmt.Errorf("error executing command:\n%s\n%w", accessDetails.GetCli(), err)
	}

	return nil
}
