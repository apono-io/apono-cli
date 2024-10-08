package requestloader

import (
	"context"
	"time"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/services"

	tea "github.com/charmbracelet/bubbletea"
)

func getUpdatedRequest(ctx context.Context, client *aponoapi.AponoClient, requestID string) tea.Cmd {
	return func() tea.Msg {
		resp, _, err := client.ClientAPI.AccessRequestsAPI.GetAccessRequest(ctx, requestID).Execute()
		if err != nil {
			return errMsg{err}
		}
		return updatedRequestMsg(*resp)
	}
}

func waitForRequest(ctx context.Context, client *aponoapi.AponoClient, creationTime time.Time, timeout time.Duration) tea.Cmd {
	return func() tea.Msg {
		newAccessRequest, err := services.WaitForNewRequest(ctx, client, creationTime, timeout)
		if err != nil {
			return errMsg{err}
		}

		return updatedRequestMsg(*newAccessRequest)
	}
}

func shouldRetryLoading(lastRequestTime time.Time, interval time.Duration) bool {
	return time.Now().After(lastRequestTime.Add(interval))
}

func ShouldStopLoading(request *clientapi.AccessRequestClientModel) bool {
	switch request.Status.Status {
	case services.AccessRequestActiveStatus, services.AccessRequestFailedStatus, services.AccessRequestRejectedStatus:
		return true

	case services.AccessRequestPendingStatus:
		if services.IsRequestWaitingForHumanApproval(request) {
			return true
		}
	case services.AccessRequestPendingMFAStatus:
		return true
	}

	return false
}
