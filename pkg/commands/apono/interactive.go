package apono

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/apono-io/apono-cli/pkg/interactive/selectors"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/banner"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/config"
	"github.com/apono-io/apono-cli/pkg/interactive/flows"
	requestloader "github.com/apono-io/apono-cli/pkg/interactive/inputs/request_loader"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/utils"
	"github.com/apono-io/apono-cli/pkg/version"

	"github.com/gookit/color"
	"github.com/spf13/cobra"
)

const (
	requestWaitTime = 90 * time.Second
)

func startMainInteractiveFlow(cmd *cobra.Command, client *aponoapi.AponoClient) error {
	_ = displayBanner(cmd) // Ignore error - banner is non-critical

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

func displayBanner(cmd *cobra.Command) error {
	ctx := cmd.Context()

	versionInfo, err := version.GetVersion(ctx)
	if err != nil || versionInfo == nil {
		versionInfo = &version.VersionInfo{
			Version:   "unknown",
			Commit:    "unknown",
			BuildDate: "unknown",
		}
	}

	profileName, _ := cmd.Flags().GetString("profile")
	if profileName == "" {
		cfg, cfgErr := config.Get()
		if cfgErr == nil && cfg.Auth.ActiveProfile != "" {
			profileName = string(cfg.Auth.ActiveProfile)
		} else {
			profileName = "default"
		}
	}

	sessionInfo, err := fetchSessionInfo(ctx)
	if err != nil {
		if isAuthError(err) {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "\n%s Authentication expired. Please run: %s\n\n",
				color.Yellow.Sprint("⚠"),
				color.Cyan.Sprint("apono login"))
		}
		sessionInfo = getCachedSessionInfo(ctx)
	}

	return banner.Display(cmd.OutOrStdout(), versionInfo, sessionInfo, profileName)
}

func fetchSessionInfo(ctx context.Context) (*banner.UserSessionInfo, error) {
	client, err := aponoapi.GetClient(ctx)
	if err != nil || client == nil {
		return nil, fmt.Errorf("no client available")
	}

	userSession, resp, err := client.ClientAPI.UserSessionAPI.GetUserSession(ctx).Execute()
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	return &banner.UserSessionInfo{
		AccountID:   userSession.Account.Id,
		AccountName: userSession.Account.Name,
		UserID:      userSession.User.Id,
		UserName:    userSession.User.Name,
		UserEmail:   userSession.User.Email,
	}, nil
}

func getCachedSessionInfo(ctx context.Context) *banner.UserSessionInfo {
	profile, err := config.GetCurrentProfile(ctx)
	if err != nil || profile == nil {
		return nil
	}

	return &banner.UserSessionInfo{
		AccountID:   profile.AccountID,
		AccountName: profile.AccountName,
		UserID:      profile.UserID,
		UserName:    profile.UserName,
		UserEmail:   profile.UserEmail,
	}
}

func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "401") ||
		strings.Contains(errMsg, "unauthorized") ||
		strings.Contains(errMsg, "authentication") ||
		strings.Contains(errMsg, "forbidden")
}
