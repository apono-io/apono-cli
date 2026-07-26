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

func FetchClients(ctx context.Context, apiClient *aponoapi.AponoClient, sessionID string) (*ClientFetchResult, error) {
	details, err := FetchAccessDetails(ctx, apiClient, sessionID)
	if err != nil {
		return nil, err
	}
	return BuildClientFetchResult(details), nil
}
