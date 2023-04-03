package commands

import (
	"context"
	"fmt"
	"strings"

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
			client, err := aponoapi.CreateClient(cmd.Context(), "default")
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
	resp, err := client.GetAccessRequestWithResponse(cmd.Context(), requestID)
	if err != nil {
		return err
	}

	accessRequest := resp.JSON200
	return printAccessRequestDetails(cmd, client, accessRequest)
}

func printAccessRequestDetails(cmd *cobra.Command, client *aponoapi.AponoClient, accessRequest *aponoapi.AccessRequest) error {
	integrationID := accessRequest.IntegrationId
	integrationResp, err := client.GetIntegrationV2WithResponse(cmd.Context(), integrationID)
	if err != nil {
		return err
	}

	table := uitable.New()
	table.Wrap = true
	table.AddRow("ID:", accessRequest.FriendlyRequestId)
	table.AddRow("Status:", coloredStatus(accessRequest.Status))
	table.AddRow("Integration:", integrationResp.JSON200.Name)
	table.AddRow("Resources:", strings.Join(accessRequest.ResourceIds, ", "))
	table.AddRow("Permissions:", strings.Join(accessRequest.Permissions, ", "))
	table.AddRow("Justification:", accessRequest.Justification)

	_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
	return err
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

func coloredStatus(status aponoapi.AccessStatusModel) string {
	statusTitle := cases.Title(language.English).String(string(status))
	switch status {
	case aponoapi.PENDING:
		return color.Yellow.Sprint(statusTitle)
	case aponoapi.APPROVED:
		return color.HiYellow.Sprint(statusTitle)
	case aponoapi.GRANTED:
		return color.Green.Sprint(statusTitle)
	case aponoapi.REJECTED, aponoapi.REVOKING, aponoapi.EXPIRED:
		return color.Gray.Sprint(statusTitle)
	case aponoapi.FAILED:
		return color.Red.Sprint(statusTitle)
	default:
		return statusTitle
	}
}
