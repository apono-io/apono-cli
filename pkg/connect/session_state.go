package connect

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
)

const SessionStatusDisconnected = "disconnected"

func FetchSession(ctx context.Context, apiClient *aponoapi.AponoClient, sessionID string) (*clientapi.AccessSessionClientModel, error) {
	session, resp, err := apiClient.ClientAPI.AccessSessionsAPI.GetAccessSession(ctx, sessionID).Execute()
	if err != nil {
		if isNotFoundResponse(resp) {
			return nil, &aponoapi.AccessSessionUnavailableError{
				Reason:     aponoapi.AccessSessionReasonNotFound,
				SessionID:  sessionID,
				HTTPStatus: http.StatusNotFound,
			}
		}
		return nil, aponoapi.DescribeAPIError(err, resp)
	}

	if isDisconnectedSession(session) {
		return nil, &aponoapi.AccessSessionUnavailableError{
			Reason:      aponoapi.AccessSessionReasonNotActive,
			SessionID:   sessionID,
			SessionName: session.Name,
		}
	}

	return session, nil
}

func ClassifyAccessDetailsError(ctx context.Context, apiClient *aponoapi.AponoClient, sessionID string, err error, resp *http.Response) error {
	if !isNotFoundResponse(resp) {
		return aponoapi.DescribeAPIError(err, resp)
	}

	unavailableErr := &aponoapi.AccessSessionUnavailableError{
		Reason:     aponoapi.AccessSessionReasonUnknown,
		SessionID:  sessionID,
		HTTPStatus: http.StatusNotFound,
	}

	session, sessionResp, sessionErr := apiClient.ClientAPI.AccessSessionsAPI.GetAccessSession(ctx, sessionID).Execute()
	switch {
	case isUnclassifiableRefetchError(sessionErr):
		return sessionErr
	case sessionErr != nil && isNotFoundResponse(sessionResp):
		unavailableErr.Reason = aponoapi.AccessSessionReasonNotFound
	case sessionErr == nil && isDisconnectedSession(session):
		unavailableErr.Reason = aponoapi.AccessSessionReasonNotActive
		unavailableErr.SessionName = session.Name
	}

	return unavailableErr
}

func isUnclassifiableRefetchError(err error) bool {
	if err == nil {
		return false
	}
	if _, isUnsupportedAccount := aponoapi.AsUnsupportedAccountError(err); isUnsupportedAccount {
		return true
	}
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func isNotFoundResponse(resp *http.Response) bool {
	return resp != nil && resp.StatusCode == http.StatusNotFound
}

func isDisconnectedSession(session *clientapi.AccessSessionClientModel) bool {
	return session != nil && strings.EqualFold(session.Status, SessionStatusDisconnected)
}
