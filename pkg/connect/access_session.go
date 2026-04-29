package connect

import (
	"context"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/utils"
)

// consumedByAponoCli is the BE wire value identifying this CLI as the
// session's credential consumer. Anything else echoed back in the response
// means another surface (e.g. the portal's Access Details dialog) used the
// creds first and the user must reset before we can launch.
const consumedByAponoCli = "consumedByAponoCli"

type ClientFetchResult struct {
	Clients    []clientapi.LauncherClientModel
	ConsumedBy string
}

func fetchClients(ctx context.Context, apiClient *aponoapi.AponoClient, sessionID string) (*ClientFetchResult, error) {
	details, _, err := apiClient.ClientAPI.AccessSessionsAPI.
		GetAccessSessionAccessDetails(ctx, sessionID).
		ConsumedBy(consumedByAponoCli).
		Execute()
	if err != nil {
		return nil, err
	}
	return &ClientFetchResult{
		Clients:    details.Launchers,
		ConsumedBy: utils.FromNullableString(details.ConsumedBy),
	}, nil
}
