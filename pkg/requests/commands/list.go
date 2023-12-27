package commands

import (
	"fmt"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/requests/utils"
	"strings"

	"github.com/apono-io/apono-sdk-go"

	"github.com/gosuri/uitable"
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

func printAccessRequestDetails(cmd *cobra.Command, client *aponoapi.AponoClient, accessRequest *apono.AccessRequest) error {
	integrationID := accessRequest.IntegrationId
	integrationResp, _, err := client.IntegrationsApi.GetIntegrationV2(cmd.Context(), integrationID).Execute()
	if err != nil {
		return err
	}

	table := uitable.New()
	table.Wrap = true
	table.AddRow("ID:", accessRequest.FriendlyRequestId)
	table.AddRow("Status:", accessRequest.Status)
	table.AddRow("Integration:", integrationResp.Name)
	table.AddRow("Resources:", strings.Join(accessRequest.ResourceIds, ", "))
	table.AddRow("Permissions:", strings.Join(accessRequest.Permissions, ", "))
	table.AddRow("Justification:", accessRequest.Justification)

	_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(cmd.OutOrStdout(), "\nYou can use the following command to make this request again:\n%s\n",
		buildNewRequestCommand(accessRequest))
	return err
}

func buildNewRequestCommand(accessRequest *apono.AccessRequest) string {
	return fmt.Sprintf("apono request -i %s -r %s -p %s -j \"%s\"",
		accessRequest.IntegrationId, strings.Join(accessRequest.ResourceIds, ","),
		strings.Join(accessRequest.Permissions, ","), accessRequest.Justification)
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
