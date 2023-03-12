package commands

import (
	"fmt"
	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
	"strings"
)

func List() *cobra.Command {
	var daysOffset int64

	cmd := &cobra.Command{
		Use:     "requests",
		GroupID: Group.ID,
		Short:   "List all access request",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.CreateClient(cmd.Context(), "default")
			if err != nil {
				return err
			}

			resp, err := client.ListAccessRequestsWithResponse(cmd.Context(), &aponoapi.ListAccessRequestsParams{
				DaysOffset: &daysOffset,
				UserId:     &client.Session.UserID,
			})
			if err != nil {
				return err
			}

			table := uitable.New()
			table.AddRow("REQUEST ID", "INTEGRATION", "RESOURCES", "PERMISSIONS", "STATUS")
			for _, ar := range resp.JSON200.Data {
				table.AddRow(ar.FriendlyRequestId, ar.IntegrationId, strings.Join(ar.ResourceIds, ", "), strings.Join(ar.Permissions, ", "), ar.Status)
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
			return err
		},
	}

	flags := cmd.Flags()
	flags.Int64VarP(&daysOffset, "days", "d", 7, "number of days to list")

	return cmd
}
