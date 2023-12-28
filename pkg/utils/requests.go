package utils

import (
	"context"
	"strings"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"

	"github.com/gookit/color"
	"github.com/gosuri/uitable"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func GenerateRequestsTable(requests []clientapi.AccessRequestClientModel) *uitable.Table {
	table := uitable.New()
	table.AddRow("REQUEST ID", "CREATED", "REVOKED", "INTEGRATIONS", "JUSTIFICATION", "STATUS")
	for _, request := range requests {
		var requestIntegrations []string
		for _, accessGroup := range request.AccessGroups {
			requestIntegrations = append(requestIntegrations, accessGroup.Integration.Name)
		}

		integrations := strings.Join(requestIntegrations, ", ")
		if integrations == "" {
			integrations = "NA"
		}

		creationTime := ConvertUnixTimeToTime(request.CreationTime)
		var revocationTime string
		if request.RevocationTime.IsSet() {
			revocationTime = DisplayTime(ConvertUnixTimeToTime(*request.RevocationTime.Get()))
		} else {
			revocationTime = "NA"
		}

		table.AddRow(request.Id, DisplayTime(creationTime), revocationTime, integrations, *request.Justification.Get(), coloredStatus(request.Status.Status))
	}

	return table
}

func GetUserLastRequest(ctx context.Context, client *aponoapi.AponoClient) (*clientapi.AccessRequestClientModel, error) {
	userLastRequests, _, err := client.ClientAPI.AccessRequestsAPI.ListAccessRequests(ctx).
		Scope(clientapi.ACCESSREQUESTSSCOPEMODEL_MY_REQUESTS).
		Limit(1).
		Execute()
	if err != nil {
		return nil, err
	}

	if len(userLastRequests.Data) == 0 {
		return nil, nil
	}

	return &userLastRequests.Data[0], nil
}

func ListRequests(ctx context.Context, client *aponoapi.AponoClient, daysOffset int64) ([]clientapi.AccessRequestClientModel, error) {
	var resultRequests []clientapi.AccessRequestClientModel

	skip := 0
	hasNextPage := true
	for ok := true; ok; ok = hasNextPage {
		resp, _, err := client.ClientAPI.AccessRequestsAPI.ListAccessRequests(ctx).
			Scope(clientapi.ACCESSREQUESTSSCOPEMODEL_MY_REQUESTS).
			Skip(int32(skip)).
			Execute()
		if err != nil {
			return nil, err
		}

		for _, request := range resp.Data {
			if isDateAfterDaysOffset(ConvertUnixTimeToTime(request.CreationTime), daysOffset) {
				resultRequests = append(resultRequests, request)
			}
		}

		skip += len(resp.Data)
		hasNextPage = int(resp.Pagination.Limit) <= len(resp.Data) && len(resultRequests) == skip
	}

	return resultRequests, nil
}

func coloredStatus(status clientapi.AccessStatus) string {
	statusTitle := cases.Title(language.English).String(string(status))
	switch status {
	case clientapi.ACCESSSTATUS_PENDING:
		return color.Yellow.Sprint(statusTitle)
	case clientapi.ACCESSSTATUS_APPROVED:
		return color.HiYellow.Sprint(statusTitle)
	case clientapi.ACCESSSTATUS_GRANTED:
		return color.Green.Sprint(statusTitle)
	case clientapi.ACCESSSTATUS_REJECTED, clientapi.ACCESSSTATUS_REVOKING, clientapi.ACCESSSTATUS_EXPIRED:
		return color.Gray.Sprint(statusTitle)
	case clientapi.ACCESSSTATUS_FAILED:
		return color.Red.Sprint(statusTitle)
	default:
		return statusTitle
	}
}
