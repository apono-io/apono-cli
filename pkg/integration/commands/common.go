package commands

import (
	"context"
	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

func listSelectableIntegrations(ctx context.Context, client *aponoapi.AponoClient) ([]aponoapi.SelectableIntegration, error) {
	resp, err := client.GetSelectableIntegrationsWithResponse(ctx, &aponoapi.GetSelectableIntegrationsParams{
		UserId: &client.Session.UserID,
	})
	if err != nil {
		return nil, err
	}

	return resp.JSON200.Data, nil
}

func listSelectablePermissions(ctx context.Context, client *aponoapi.AponoClient, integrationId string) ([]string, error) {
	resp, err := client.GetSelectablePermissionsWithResponse(ctx, integrationId, &aponoapi.GetSelectablePermissionsParams{
		UserId: &client.Session.UserID,
	})
	if err != nil {
		return nil, err
	}

	return resp.JSON200.Data, nil
}

func listSelectableResources(ctx context.Context, client *aponoapi.AponoClient, integrationId string) ([]aponoapi.SelectableResource, error) {
	resp, err := client.GetSelectableResourcesWithResponse(ctx, integrationId, &aponoapi.GetSelectableResourcesParams{
		UserId: &client.Session.UserID,
	})
	if err != nil {
		return nil, err
	}

	return resp.JSON200.Data, nil
}

func getIntegration(ctx context.Context, client *aponoapi.AponoClient, integrationId string) (*aponoapi.Integration, error) {
	resp, err := client.GetIntegrationV2WithResponse(ctx, integrationId)
	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}
