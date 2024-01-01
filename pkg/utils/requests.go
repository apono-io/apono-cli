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
	for {
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

		hasNextPage := int(resp.Pagination.Limit) <= len(resp.Data) && len(resultRequests) == skip
		if !hasNextPage {
			break
		}
	}

	return resultRequests, nil
}

func ListAccessRequestAccessUnits(ctx context.Context, client *aponoapi.AponoClient, requestID string) ([]clientapi.AccessUnitClientModel, error) {
	accessRequest, _, err := client.ClientAPI.AccessRequestsAPI.GetAccessRequest(ctx, requestID).Execute()
	if err != nil {
		return nil, err
	}

	var requestAccessUnits []clientapi.AccessUnitClientModel
	for _, accessGroup := range accessRequest.AccessGroups {
		groupAccessUnits, err := listAccessGroupAccessUnits(ctx, client, accessGroup.Id)
		if err != nil {
			return nil, err
		}

		requestAccessUnits = append(requestAccessUnits, groupAccessUnits...)
	}

	return requestAccessUnits, nil
}

func listAccessGroupAccessUnits(ctx context.Context, client *aponoapi.AponoClient, accessGroupID string) ([]clientapi.AccessUnitClientModel, error) {
	return getAllPages(ctx, client, func(ctx context.Context, client *aponoapi.AponoClient, skip int32) ([]clientapi.AccessUnitClientModel, *clientapi.PaginationClientInfoModel, error) {
		resp, _, err := client.ClientAPI.AccessGroupsAPI.GetAccessGroupUnits(ctx, accessGroupID).
			Skip(skip).
			Execute()
		if err != nil {
			return nil, nil, err
		}

		return resp.Data, &resp.Pagination, nil
	})
}

func RevokeRequest(ctx context.Context, client *aponoapi.AponoClient, requestID string) error {
	_, resp, err := client.ClientAPI.AccessRequestsAPI.RevokeAccessRequest(ctx, requestID).Execute()
	if resp != nil {
		apiError := returnAPIResponseError(resp)
		if apiError != nil {
			return apiError
		}
	}

	return err
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
