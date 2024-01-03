package actions

import (
	"fmt"

	"github.com/apono-io/apono-cli/pkg/services"

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

			return listRequestsOutput(cmd, client, daysOffset)
		},
	}

	flags := cmd.Flags()
	flags.Int64VarP(&daysOffset, "days", "d", 7, "number of days to list")

	return cmd
}

func listRequestsOutput(cmd *cobra.Command, client *aponoapi.AponoClient, daysOffset int64) error {
	requests, err := services.ListRequests(cmd.Context(), client, daysOffset)
	if err != nil {
		return err
	}

	table := services.GenerateRequestsTable(requests)

	_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
	return err
}
