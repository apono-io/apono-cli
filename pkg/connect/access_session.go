package connect

import (
	"context"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/utils"
)

const CliClientID = "cli"

type ClientFetchResult struct {
	Clients    []clientapi.LauncherClientModel
	ConsumedBy string
}

// FetchAccessDetails performs the single credential-consuming access-details
// fetch for a session. The backend hands out the one-time password on the first
// call and deletes it, so this must be called exactly once per connect and its
// result reused for everything that follows.
func FetchAccessDetails(ctx context.Context, apiClient *aponoapi.AponoClient, sessionID string) (*clientapi.AccessSessionDetailsClientModel, error) {
	details, _, err := apiClient.ClientAPI.AccessSessionsAPI.
		GetAccessSessionAccessDetails(ctx, sessionID).
		ConsumedBy(aponoapi.ConsumedByAponoCli).
		Execute()
	if err != nil {
		return nil, err
	}
	return details, nil
}

// BuildClientFetchResult projects already-fetched access details into the
// launcher list (appending the cli command as a client). It does not fetch —
// callers that already hold details reuse them here instead of consuming the
// one-time password a second time.
func BuildClientFetchResult(details *clientapi.AccessSessionDetailsClientModel) *ClientFetchResult {
	clients := details.Launchers
	if cli := utils.FromNullableString(details.Cli); cli != "" {
		clients = append(clients, clientapi.LauncherClientModel{
			Id:                CliClientID,
			LauncherType:      ClientKindCLI,
			InvocationCommand: cli,
		})
	}
	return &ClientFetchResult{
		Clients:    clients,
		ConsumedBy: utils.FromNullableString(details.ConsumedBy),
	}
}

// FetchClients fetches a session's access details and projects them into the
// launcher list. Callers that already hold details should use
// FetchAccessDetails + BuildClientFetchResult to avoid a second consuming fetch.
func FetchClients(ctx context.Context, apiClient *aponoapi.AponoClient, sessionID string) (*ClientFetchResult, error) {
	details, err := FetchAccessDetails(ctx, apiClient, sessionID)
	if err != nil {
		return nil, err
	}
	return BuildClientFetchResult(details), nil
}
