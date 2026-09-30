package apono

import (
	"context"
	"fmt"
	"time"

	"github.com/apono-io/apono-cli/pkg/interactive/selectors"

	"github.com/apono-io/apono-cli/pkg/analytics"
	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/interactive/flows"
	requestloader "github.com/apono-io/apono-cli/pkg/interactive/inputs/request_loader"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/utils"

	"github.com/gookit/color"
	"github.com/spf13/cobra"
)

const (
	requestWaitTime = 90 * time.Second
)

func startMainInteractiveFlow(cmd *cobra.Command, client *aponoapi.AponoClient) error {
	cmd.SetContext(analytics.CreateInteractiveModeContext(cmd.Context()))
	analytics.SendInteractiveSessionStartedEvent(cmd.Context())

	services.FetchAndPrintNotifications(cmd, client.ClientAPI)

	mainAction, err := selectors.RunMainActionSelector()
	if err != nil {
		return err
	}

	switch mainAction {
	case selectors.RequestAccessOption:
		analytics.SendOptionSelectedEvent(cmd.Context(), analytics.InteractiveSurface, analytics.SelectIDActionSelection, analytics.RequestNewAccessFlow)
		return RunFullRequestInteractiveFlow(cmd, client)
	case selectors.ConnectOption:
		analytics.SendOptionSelectedEvent(cmd.Context(), analytics.InteractiveSurface, analytics.SelectIDActionSelection, analytics.ConnectToResourceFlow)
		return flows.RunUseSessionInteractiveFlow(cmd, client, "")

	default:
		return fmt.Errorf("unknown option selected: %s", mainAction)
	}
}

func RunFullRequestInteractiveFlow(cmd *cobra.Command, client *aponoapi.AponoClient) error {
	req, reqModel, err := flows.StartRequestBuilderInteractiveMode(cmd, client)
	if err != nil {
		return err
	}
	createResp, resp, err := client.ClientAPI.AccessRequestsAPI.CreateUserAccessRequest(cmd.Context()).
		CreateAccessRequestClientModel(*req).
		Execute()
	if err != nil {
		apiError := utils.ReturnAPIResponseError(resp)
		if apiError != nil {
			return apiError
		}
		return err
	}
	sendAccessRequestSubmittedEvent(cmd.Context(), req, reqModel)
	if len(createResp.RequestIds) == 0 {
		return fmt.Errorf("failed to create access request, no request IDs returned from the API")
	}

	requestID := createResp.RequestIds[0]
	newAccessRequest, err := requestloader.RunRequestLoader(cmd.Context(), client, requestID, requestWaitTime, false)
	if err != nil {
		return err
	}

	if newAccessRequest.Status.Status != services.AccessRequestActiveStatus {
		fmt.Println()

		err = services.PrintAccessRequests(cmd, []clientapi.AccessRequestClientModel{*newAccessRequest}, utils.TableFormat, false)
		if err != nil {
			return err
		}

		if services.IsRequestWaitingForMFA(newAccessRequest) {
			err = services.PrintAccessRequestMFALink(cmd, &newAccessRequest.Id)
			if err != nil {
				return err
			}
		}

		return nil
	}

	accessGrantedMsg := fmt.Sprintf("\nAccess request %s granted\n", color.Green.Sprintf("%s", newAccessRequest.Id))
	_, err = fmt.Fprintln(cmd.OutOrStdout(), accessGrantedMsg)
	if err != nil {
		return err
	}

	return flows.RunUseSessionInteractiveFlow(cmd, client, newAccessRequest.Id)
}

func sendAccessRequestSubmittedEvent(ctx context.Context, req *clientapi.CreateAccessRequestClientModel, reqModel *flows.CreateAccessRequestWithFullModels) {
	requestType := analytics.RequestTypeIntegration
	var integrationName string
	var bundleName string
	var resourceType string
	var resourcesCount int
	var permissionNames []string
	if len(reqModel.Bundles) > 0 {
		requestType = analytics.RequestTypeBundle
		bundleName = reqModel.Bundles[0].Name
	}
	if len(reqModel.Integrations) > 0 {
		integrationName = reqModel.Integrations[0].Name
	}
	if len(reqModel.Resources) > 0 {
		resourceType = reqModel.Resources[0].Name
		resourcesCount = len(reqModel.Resources)
	}
	for _, permission := range reqModel.Permissions {
		permissionNames = append(permissionNames, permission.Name)
	}
	analytics.SendAccessRequestSubmittedEvent(ctx, analytics.AccessRequestSubmittedProperties{
		RequestType:     requestType,
		BundleName:      bundleName,
		IntegrationName: integrationName,
		ResourceType:    resourceType,
		ResourcesCount:  resourcesCount,
		Permissions:     permissionNames,
		Duration:        reqModel.Duration,
		Justification:   req.GetJustification(),
	})
}
