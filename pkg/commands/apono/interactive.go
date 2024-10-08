package apono

import (
	"fmt"
	"time"

	"github.com/apono-io/apono-cli/pkg/interactive/selectors"

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
	mainAction, err := selectors.RunMainActionSelector()
	if err != nil {
		return err
	}

	switch mainAction {
	case selectors.RequestAccessOption:
		return RunFullRequestInteractiveFlow(cmd, client)
	case selectors.ConnectOption:
		return flows.RunUseSessionInteractiveFlow(cmd, client, "")

	default:
		return fmt.Errorf("unknown option selected: %s", mainAction)
	}
}

func RunFullRequestInteractiveFlow(cmd *cobra.Command, client *aponoapi.AponoClient) error {
	req, err := flows.StartRequestBuilderInteractiveMode(cmd, client)
	if err != nil {
		return err
	}

	creationTime := time.Now()

	_, resp, err := client.ClientAPI.AccessRequestsAPI.CreateUserAccessRequest(cmd.Context()).
		CreateAccessRequestClientModel(*req).
		Execute()
	if err != nil {
		apiError := utils.ReturnAPIResponseError(resp)
		if apiError != nil {
			return apiError
		}

		return err
	}

	newAccessRequest, err := requestloader.RunRequestLoader(cmd.Context(), client, creationTime, requestWaitTime, false)
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

	accessGrantedMsg := fmt.Sprintf("\nAccess request %s granted\n", color.Green.Sprintf(newAccessRequest.Id))
	_, err = fmt.Fprintln(cmd.OutOrStdout(), accessGrantedMsg)
	if err != nil {
		return err
	}

	return flows.RunUseSessionInteractiveFlow(cmd, client, newAccessRequest.Id)
}
