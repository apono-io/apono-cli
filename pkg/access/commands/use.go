package commands

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

const outputFlagName = "output"

func AccessDetails() *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:     "use [id]",
		GroupID: Group.ID,
		Short:   "Get access session details",
		Args:    cobra.ExactArgs(1), // This will enforce that exactly 1 argument must be provided
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			id := args[0]
			accessDetails, _, err := client.ClientAPI.AccessSessionsAPI.GetAccessSessionAccessDetails(cmd.Context(), id).Execute()
			if err != nil {
				return err
			}

			var output string
			switch outputFormat {
			case "cli":
				link := accessDetails.GetCli()
				if accessDetails.GetCli() == "" {
					return errors.New("no output for format: cli")
				}
				output = link
			case "link":
				link := accessDetails.GetLink()
				if link.GetUrl() == "" {
					return errors.New("no output for format: link")
				}
				output = link.GetUrl()
			case "instructions":
				if accessDetails.Instructions.Plain == "" {
					return errors.New("no output for format: instructions")
				}
				output = accessDetails.Instructions.Plain
			case "json":
				if accessDetails.Json == nil {
					return errors.New("no output for format: json")
				}
				var outputBytes []byte
				outputBytes, err = json.Marshal(accessDetails.Credentials)
				if err != nil {
					return err
				}
				output = string(outputBytes)
			}

			if output == "" {
				return fmt.Errorf("no output for format: %s", outputFormat)
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

	return cmd
}
