package commands

import (
	"fmt"

	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/utils"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

func List() *cobra.Command {
	var daysOffset int64

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all access request",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			return showRequestsSummary(cmd, client, daysOffset)
		},
	}

	flags := cmd.Flags()
	flags.Int64VarP(&daysOffset, "days", "d", 7, "number of days to list")

	return cmd
}

func showRequestStatus(cmd *cobra.Command, client *aponoapi.AponoClient, requestID string) error {
	resp, _, err := client.ClientAPI.AccessRequestsAPI.GetAccessRequest(cmd.Context(), requestID).Execute()
	if err != nil {
		return err
	}

	table := utils.GenerateRequestsTable([]clientapi.AccessRequestClientModel{*resp})

	_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
	return err
}

func showRequestsSummary(cmd *cobra.Command, client *aponoapi.AponoClient, daysOffset int64) error {
	requests, err := utils.ListRequests(cmd.Context(), client, daysOffset)
	if err != nil {
		return err
	}

	table := utils.GenerateRequestsTable(requests)

	_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
	return err
}
