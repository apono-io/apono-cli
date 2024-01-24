package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/apono-io/apono-cli/pkg/utils"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"

	"github.com/gookit/color"
	"github.com/gosuri/uitable"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	AccessRequestInitStatus               = "Initializing"
	AccessRequestPendingStatus            = "Pending"
	AccessRequestGrantingStatus           = "Granting"
	AccessRequestRejectedStatus           = "Rejected"
	AccessRequestActiveStatus             = "Active"
	AccessRequestRevokingStatus           = "Revoking"
	AccessRequestRevokedStatus            = "Revoked"
	AccessRequestFailedStatus             = "Failed"
	AccessRequestWaitingForApprovalStatus = "Pending Approval"
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

		creationTime := utils.ConvertUnixTimeToTime(request.CreationTime)
		var revocationTime string
		if request.RevocationTime.IsSet() {
			revocationTime = utils.DisplayTime(utils.ConvertUnixTimeToTime(*request.RevocationTime.Get()))
		} else {
			revocationTime = "NA"
		}

		table.AddRow(request.Id, utils.DisplayTime(creationTime), revocationTime, integrations, *request.Justification.Get(), ColoredStatus(request))
	}

	return table
}

func WaitForNewRequest(ctx context.Context, client *aponoapi.AponoClient, creationTime time.Time, timeout time.Duration) (*clientapi.AccessRequestClientModel, error) {
	startTime := time.Now()
	for {
		lastRequest, err := getUserLastRequest(ctx, client)
		if err != nil {
			return nil, err
		}

		lastRequestCreationTime := utils.ConvertUnixTimeToTime(lastRequest.CreationTime)
		if lastRequestCreationTime.After(creationTime) {
			return lastRequest, nil
		}

		time.Sleep(1 * time.Second)

		if time.Now().After(startTime.Add(timeout)) {
			return nil, fmt.Errorf("timeout while waiting for request to be created")
		}
	}
}

func getUserLastRequest(ctx context.Context, client *aponoapi.AponoClient) (*clientapi.AccessRequestClientModel, error) {
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
			if utils.IsDateAfterDaysOffset(utils.ConvertUnixTimeToTime(request.CreationTime), daysOffset) {
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

func IsRequestWaitingForHumanApproval(request *clientapi.AccessRequestClientModel) bool {
	if request.Status.Status != AccessRequestPendingStatus {
		return false
	}

	if !request.Challenge.IsSet() {
		return false
	}

	if len(request.Challenge.Get().Approvers) == 0 {
		return false
	}

	return true
}

func listAccessGroupAccessUnits(ctx context.Context, client *aponoapi.AponoClient, accessGroupID string) ([]clientapi.AccessUnitClientModel, error) {
	return utils.GetAllPages(ctx, client, func(ctx context.Context, client *aponoapi.AponoClient, skip int32) ([]clientapi.AccessUnitClientModel, *clientapi.PaginationClientInfoModel, error) {
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
		apiError := utils.ReturnAPIResponseError(resp)
		if apiError != nil {
			return apiError
		}
	}

	return err
}

func ColoredStatus(request clientapi.AccessRequestClientModel) string {
	status := request.Status.Status
	if IsRequestWaitingForHumanApproval(&request) {
		status = AccessRequestWaitingForApprovalStatus
	}

	statusTitle := cases.Title(language.English).String(status)
	switch status {
	case AccessRequestWaitingForApprovalStatus:
		return color.HiYellow.Sprint(statusTitle)
	case AccessRequestInitStatus, AccessRequestPendingStatus:
		return color.Yellow.Sprint(statusTitle)
	case AccessRequestGrantingStatus:
		return color.HiYellow.Sprint(statusTitle)
	case AccessRequestActiveStatus:
		return color.Green.Sprint(statusTitle)
	case AccessRequestRevokingStatus, AccessRequestRevokedStatus, AccessRequestRejectedStatus:
		return color.Gray.Sprint(statusTitle)
	case AccessRequestFailedStatus:
		return color.Red.Sprint(statusTitle)
	default:
		return statusTitle
	}
}

func GetEmptyNewRequestAPIModel() *clientapi.CreateAccessRequestClientModel {
	return &clientapi.CreateAccessRequestClientModel{
		FilterBundleIds:       []string{},
		FilterIntegrationIds:  []string{},
		FilterResourceTypeIds: []string{},
		FilterResourceIds:     []string{},
		FilterPermissionIds:   []string{},
		FilterAccessUnitIds:   []string{},
	}
}
