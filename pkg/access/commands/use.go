package commands

import (
	"encoding/json"
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
				output = accessDetails.GetCli()
			case "link":
				link := accessDetails.GetLink()
				output = link.GetUrl()
			case "instructions":
				output = accessDetails.Instructions
			case "json":
				outputBytes, _ := json.Marshal(accessDetails)
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

	return cmd
}
