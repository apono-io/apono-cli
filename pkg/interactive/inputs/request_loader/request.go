package requestloader

import (
	"context"
	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/services"
	tea "github.com/charmbracelet/bubbletea"
	"time"
)

func getUpdatedRequest(ctx context.Context, client *aponoapi.AponoClient, requestID string) tea.Cmd {
	return func() tea.Msg {
		resp, _, err := client.ClientAPI.AccessRequestsAPI.GetAccessRequest(ctx, requestID).Execute()
		if err != nil {
			return errMsg{err}
		}
		return statusMsg(*resp)
	}
}

func waitForRequest(ctx context.Context, client *aponoapi.AponoClient, creationTime time.Time, timeout time.Duration) tea.Cmd {
	return func() tea.Msg {
		newAccessRequest, err := services.WaitForNewRequest(ctx, client, creationTime, timeout)
		if err != nil {
			return errMsg{err}
		}

		return statusMsg(*newAccessRequest)
	}
}

func shouldStopLoading(request *clientapi.AccessRequestClientModel) bool {
	switch request.Status.Status {
	case clientapi.ACCESSSTATUS_FAILED, clientapi.ACCESSSTATUS_GRANTED:
		return true

	case clientapi.ACCESSSTATUS_PENDING:
		if services.IsRequestWaitingForApproval(request) {
			return true
		}
	}

	return false
}
