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

func fetchClients(ctx context.Context, apiClient *aponoapi.AponoClient, sessionID string) (*ClientFetchResult, error) {
	details, _, err := apiClient.ClientAPI.AccessSessionsAPI.
		GetAccessSessionAccessDetails(ctx, sessionID).
		ConsumedBy(aponoapi.ConsumedByAponoCli).
		Execute()
	if err != nil {
		return nil, err
	}
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
	}, nil
}
