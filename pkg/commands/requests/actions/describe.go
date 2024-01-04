package actions

import (
	"fmt"
	"os"

	"github.com/apono-io/apono-cli/pkg/services"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/clientapi"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

func Describe() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "describe <request_id>",
		Short:   "Return the details for the specified access request",
		Aliases: []string{"get"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				err := cmd.Help()
				if err != nil {
					return err
				}

				os.Exit(0)
			}

			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			requestID := args[0]
			return getRequestOutput(cmd, client, requestID)
		},
	}

	return cmd
}

func getRequestOutput(cmd *cobra.Command, client *aponoapi.AponoClient, requestID string) error {
	resp, _, err := client.ClientAPI.AccessRequestsAPI.GetAccessRequest(cmd.Context(), requestID).Execute()
	if err != nil {
		return err
	}

	table := services.GenerateRequestsTable([]clientapi.AccessRequestClientModel{*resp})

	_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
	return err
}
