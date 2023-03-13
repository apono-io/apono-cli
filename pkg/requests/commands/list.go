package commands

import (
	"context"
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

			requests, err := listRequests(cmd.Context(), client, daysOffset)
			if err != nil {
				return err
			}

			integrations, err := listIntegrations(cmd.Context(), client)
			if err != nil {
				return err
			}

			table := uitable.New()
			table.AddRow("REQUEST ID", "INTEGRATION", "RESOURCES", "PERMISSIONS", "STATUS")
			for _, request := range requests {
				integrationId := request.IntegrationId
				integration := integrationId
				if integrationName, found := integrations[integrationId]; found {
					integration = integrationName
				}

				resourceIds := strings.Join(request.ResourceIds, ", ")
				permissions := strings.Join(request.Permissions, ", ")
				table.AddRow(request.FriendlyRequestId, integration, resourceIds, permissions, request.Status)
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
			return err
		},
	}

	flags := cmd.Flags()
	flags.Int64VarP(&daysOffset, "days", "d", 7, "number of days to list")

	return cmd
}

func listRequests(ctx context.Context, client *aponoapi.AponoClient, daysOffset int64) ([]aponoapi.AccessRequest, error) {
	resp, err := client.ListAccessRequestsWithResponse(ctx, &aponoapi.ListAccessRequestsParams{
		DaysOffset: &daysOffset,
		UserId:     &client.Session.UserID,
	})
	if err != nil {
		return nil, err
	}

	return resp.JSON200.Data, nil
}

func listIntegrations(ctx context.Context, client *aponoapi.AponoClient) (map[string]string, error) {
	resp, err := client.ListIntegrationsV2WithResponse(ctx)
	if err != nil {
		return nil, err
	}

	data := resp.JSON200.Data
	integrations := make(map[string]string)
	for _, integration := range data {
		integrations[integration.Id] = integration.Name
	}

	return integrations, nil
}
