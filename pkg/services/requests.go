package services

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/apono-io/apono-cli/pkg/config"
	"github.com/apono-io/apono-cli/pkg/styles"

	"github.com/apono-io/apono-cli/pkg/utils"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"

	"github.com/gookit/color"
	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	AccessRequestInitStatus               = "Initializing"
	AccessRequestPendingStatus            = "Pending"
	AccessRequestPendingMFAStatus         = "PendingMFA"
	AccessRequestGrantingStatus           = "Granting"
	AccessRequestRejectedStatus           = "Rejected"
	AccessRequestActiveStatus             = "Active"
	AccessRequestRevokingStatus           = "Revoking"
	AccessRequestRevokedStatus            = "Revoked"
	AccessRequestFailedStatus             = "Failed"
	AccessRequestWaitingForApprovalStatus = "Pending Approval"
	AccessRequestWaitingForMFAStatus      = "Pending MFA"

	dryRunFieldMissionCode               = "FIELD_MISSING"
	dryRunJustificationFieldName         = "justification"
	dryRunInvalidDurationCode            = "INVALID_DURATION"
	dryRunDurationInSecFieldName         = "duration_in_sec"
	dryRunMaxDurationInSecondsDetailsKey = "max_duration"

	maxRequestDuration = math.MaxInt32 * time.Second
)

func PrintAccessRequests(cmd *cobra.Command, requests []clientapi.AccessRequestClientModel, format utils.Format, printAsArray bool) error {
	switch format {
	case utils.TableFormat:
		table := generateRequestsTable(requests)

		_, err := fmt.Fprintln(cmd.OutOrStdout(), table)
		return err
	case utils.JSONFormat:
		if printAsArray {
			return utils.PrintObjectsAsJSON(cmd.OutOrStdout(), requests)
		} else {
			return utils.PrintObjectsAsJSON(cmd.OutOrStdout(), requests[0])
		}
	case utils.YamlFormat:
		if printAsArray {
			return utils.PrintObjectsAsYaml(cmd.OutOrStdout(), requests)
		} else {
			return utils.PrintObjectsAsYaml(cmd.OutOrStdout(), requests[0])
		}
	default:
		return fmt.Errorf("unsupported output format")
	}
}

func PrintAccessRequestMFALink(cmd *cobra.Command, requestID *string) error {
	currentConfig, err := config.GetCurrentProfile(cmd.Context())
	if err != nil {
		return err
	}

	portalURL := currentConfig.PortalURL
	if portalURL == "" {
		portalURL = config.PortalDefaultURL
	}
	link := fmt.Sprintf("%s/requests/open", portalURL)

	var prefixMessage string
	if requestID != nil {
		prefixMessage = fmt.Sprintf("Request %s", color.Bold.Sprint(*requestID))
	} else {
		prefixMessage = "Some requests"
	}

	_, err = fmt.Fprintf(
		cmd.OutOrStdout(),
		"\n%s %s requires completing MFA to proceed. It only takes a minute and helps keep you account secure: %s\n",
		styles.NoticeMsgPrefix,
		prefixMessage,
		color.Green.Sprint(link),
	)
	if err != nil {
		return err
	}

	return nil
}

func generateRequestsTable(requests []clientapi.AccessRequestClientModel) *uitable.Table {
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

func IsRequestWaitingForMFA(request *clientapi.AccessRequestClientModel) bool {
	return request.Status.Status == AccessRequestPendingMFAStatus
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

func DryRunRequest(ctx context.Context, client *aponoapi.AponoClient, request *clientapi.CreateAccessRequestClientModel) (*clientapi.DryRunClientResponse, error) {
	dryRunRequest := clientapi.CreateAccessRequestClientModel{
		FilterBundleIds:       request.FilterBundleIds,
		FilterIntegrationIds:  request.FilterIntegrationIds,
		FilterResourceTypeIds: request.FilterResourceTypeIds,
		FilterResourceIds:     request.FilterResourceIds,
		FilterResources:       request.FilterResources,
		FilterPermissionIds:   request.FilterPermissionIds,
		FilterAccessUnitIds:   request.FilterAccessUnitIds,
		Justification:         request.Justification,
		DurationInSec:         request.DurationInSec,
	}

	dryRunResponse, resp, err := client.ClientAPI.AccessRequestsAPI.DryRunCreateUserAccessRequest(ctx).
		CreateAccessRequestClientModel(dryRunRequest).
		Execute()
	if resp != nil {
		apiError := utils.ReturnAPIResponseError(resp)
		if apiError != nil {
			return nil, apiError
		}
	}

	return dryRunResponse, err
}

func IsJustificationOptionalForRequest(dryRunResponse *clientapi.DryRunClientResponse) bool {
	if dryRunResponse == nil {
		return false
	}

	for _, requestError := range dryRunResponse.Errors {
		if requestError.Code == dryRunFieldMissionCode && requestError.Field == dryRunJustificationFieldName {
			return false
		}
	}

	return true
}

func IsDurationRequiredForRequest(dryRunResponse *clientapi.DryRunClientResponse) bool {
	if dryRunResponse == nil {
		return false
	}

	for _, requestError := range dryRunResponse.Errors {
		if requestError.Code == dryRunInvalidDurationCode && requestError.Field == dryRunDurationInSecFieldName {
			return true
		}
	}

	return false
}

func GetMaximumRequestDuration(dryRunResponse *clientapi.DryRunClientResponse) time.Duration {
	if dryRunResponse == nil {
		return maxRequestDuration
	}

	for _, requestError := range dryRunResponse.Errors {
		if requestError.Code == dryRunInvalidDurationCode && requestError.Field == dryRunDurationInSecFieldName {
			if maxDurationInSecondsDetails, ok := requestError.Details[dryRunMaxDurationInSecondsDetailsKey]; ok {
				var maxDurationInSecondsResp float64
				maxDurationInSecondsResp, ok = maxDurationInSecondsDetails.(float64)
				if ok {
					return time.Duration(maxDurationInSecondsResp) * time.Second
				}
			}
		}
	}

	return maxRequestDuration
}

func ColoredStatus(request clientapi.AccessRequestClientModel) string {
	status := request.Status.Status
	if IsRequestWaitingForHumanApproval(&request) {
		status = AccessRequestWaitingForApprovalStatus
	}
	if IsRequestWaitingForMFA(&request) {
		status = AccessRequestWaitingForMFAStatus
	}

	statusTitle := cases.Title(language.English).String(status)
	switch status {
	case AccessRequestWaitingForApprovalStatus, AccessRequestWaitingForMFAStatus:
		return color.HiYellow.Sprint(statusTitle)
	case AccessRequestInitStatus, AccessRequestPendingStatus, AccessRequestPendingMFAStatus:
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
		FilterResources:       []clientapi.ResourceFilter{},
		FilterPermissionIds:   []string{},
		FilterAccessUnitIds:   []string{},
	}
}

func ListResourceFiltersFromResourcesIDs(resourcesIDs []string) []clientapi.ResourceFilter {
	var resourceFilters []clientapi.ResourceFilter
	for _, resourceID := range resourcesIDs {
		resourceFilters = append(resourceFilters, clientapi.ResourceFilter{
			Type:  clientapi.RESOURCEFILTERTYPE_ID,
			Value: resourceID,
		})
	}

	return resourceFilters
}
