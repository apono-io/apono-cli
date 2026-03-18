package requestloader

import (
	"context"
	"time"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/services"

	tea "github.com/charmbracelet/bubbletea"
)

func getRequestByID(ctx context.Context, client *aponoapi.AponoClient, requestID string) tea.Cmd {
	return func() tea.Msg {
		resp, _, err := client.ClientAPI.AccessRequestsAPI.GetAccessRequest(ctx, requestID).Execute()
		if err != nil {
			return errMsg{err}
		}
		return updatedRequestMsg(*resp)
	}
}

func shouldRetryLoading(lastRequestTime time.Time, interval time.Duration) bool {
	return time.Now().After(lastRequestTime.Add(interval))
}

func ShouldStopLoading(request *clientapi.AccessRequestClientModel) bool {
	return services.ShouldStopWaitingForRequest(request)
}
