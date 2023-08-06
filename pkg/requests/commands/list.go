package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/apono-io/apono-sdk-go"

	"github.com/gookit/color"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

func List() *cobra.Command {
	var daysOffset int64
	var requestID string

	cmd := &cobra.Command{
		Use:     "requests",
		GroupID: Group.ID,
		Short:   "List all access request",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			if requestID != "" {
				return showRequestStatus(cmd, client, requestID)
			}

			return showRequestsSummary(cmd, client, daysOffset)
		},
	}

	flags := cmd.Flags()
	flags.Int64VarP(&daysOffset, "days", "d", 7, "number of days to list")
	flags.StringVarP(&requestID, "id", "i", "", "specific request id")

	return cmd
}

func showRequestStatus(cmd *cobra.Command, client *aponoapi.AponoClient, requestID string) error {
	resp, _, err := client.AccessRequestsApi.GetAccessRequest(cmd.Context(), requestID).Execute()
	if err != nil {
		return err
	}

	return printAccessRequestDetails(cmd, client, resp)
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
	table.AddRow("Status:", coloredStatus(accessRequest.Status))
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
		integrationID := request.IntegrationId
		integration := integrationID
		if integrationName, found := integrations[integrationID]; found {
			integration = integrationName
		}

		resourceIds := strings.Join(request.ResourceIds, ", ")
		permissions := strings.Join(request.Permissions, ", ")
		table.AddRow(request.FriendlyRequestId, integration, resourceIds, permissions, coloredStatus(request.Status))
	}

	_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
	return err
}

func listRequests(ctx context.Context, client *aponoapi.AponoClient, daysOffset int64) ([]apono.AccessRequest, error) {
	resp, _, err := client.AccessRequestsApi.ListAccessRequests(ctx).
		DaysOffset(daysOffset).
		UserId(client.Session.UserID).
		Execute()
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

func listIntegrations(ctx context.Context, client *aponoapi.AponoClient) (map[string]string, error) {
	resp, _, err := client.IntegrationsApi.ListIntegrationsV2(ctx).Execute()
	if err != nil {
		return nil, err
	}

	data := resp.Data
	integrations := make(map[string]string)
	for _, integration := range data {
		integrations[integration.Id] = integration.Name
	}

	return integrations, nil
}

func coloredStatus(status apono.AccessStatusModel) string {
	statusTitle := cases.Title(language.English).String(string(status))
	switch status {
	case apono.ACCESSSTATUSMODEL_PENDING:
		return color.Yellow.Sprint(statusTitle)
	case apono.ACCESSSTATUSMODEL_APPROVED:
		return color.HiYellow.Sprint(statusTitle)
	case apono.ACCESSSTATUSMODEL_GRANTED:
		return color.Green.Sprint(statusTitle)
	case apono.ACCESSSTATUSMODEL_REJECTED, apono.ACCESSSTATUSMODEL_REVOKING, apono.ACCESSSTATUSMODEL_EXPIRED:
		return color.Gray.Sprint(statusTitle)
	case apono.ACCESSSTATUSMODEL_FAILED:
		return color.Red.Sprint(statusTitle)
	default:
		return statusTitle
	}
}
