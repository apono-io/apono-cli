package utils

import (
	"context"
	"fmt"
	"strings"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
)

const (
	integrationNameSeparator = "/"
)

func ListIntegrations(ctx context.Context, client *aponoapi.AponoClient) ([]clientapi.IntegrationClientModel, error) {
	return getAllPages(ctx, client, func(ctx context.Context, client *aponoapi.AponoClient, skip int32) ([]clientapi.IntegrationClientModel, *clientapi.PaginationClientInfoModel, error) {
		resp, _, err := client.ClientAPI.InventoryAPI.ListIntegration(ctx).
			Skip(skip).
			Execute()
		if err != nil {
			return nil, nil, err
		}

		return resp.Data, &resp.Pagination, nil
	})
}

func ListResourceTypes(ctx context.Context, client *aponoapi.AponoClient, integrationID string) ([]clientapi.ResourceTypeClientModel, error) {
	return getAllPages(ctx, client, func(ctx context.Context, client *aponoapi.AponoClient, skip int32) ([]clientapi.ResourceTypeClientModel, *clientapi.PaginationClientInfoModel, error) {
		resp, _, err := client.ClientAPI.InventoryAPI.ListResourceTypes(ctx).
			IntegrationId(integrationID).
			Skip(skip).
			Execute()
		if err != nil {
			return nil, nil, err
		}

		return resp.Data, &resp.Pagination, nil
	})
}

func ListResources(ctx context.Context, client *aponoapi.AponoClient, integrationID string, resourceType string) ([]clientapi.ResourceClientModel, error) {
	return getAllPages(ctx, client, func(ctx context.Context, client *aponoapi.AponoClient, skip int32) ([]clientapi.ResourceClientModel, *clientapi.PaginationClientInfoModel, error) {
		resp, _, err := client.ClientAPI.InventoryAPI.ListResources(ctx).
			IntegrationId(integrationID).
			ResourceTypeId(resourceType).
			Skip(skip).
			Execute()
		if err != nil {
			return nil, nil, err
		}

		return resp.Data, &resp.Pagination, nil
	})
}

func ListPermissions(ctx context.Context, client *aponoapi.AponoClient, integrationID string, resourceType string) ([]clientapi.PermissionClientModel, error) {
	return getAllPages(ctx, client, func(ctx context.Context, client *aponoapi.AponoClient, skip int32) ([]clientapi.PermissionClientModel, *clientapi.PaginationClientInfoModel, error) {
		resp, _, err := client.ClientAPI.InventoryAPI.ListPermissions(ctx).
			IntegrationId(integrationID).
			ResourceTypeId(resourceType).
			Skip(skip).
			Execute()
		if err != nil {
			return nil, nil, err
		}

		return resp.Data, &resp.Pagination, nil
	})
}

func GetIntegrationByIDOrByTypeAndName(ctx context.Context, client *aponoapi.AponoClient, integrationIDOrName string) (*clientapi.IntegrationClientModel, error) {
	integrations, err := ListIntegrations(ctx, client)
	if err != nil {
		return nil, err
	}

	var integration *clientapi.IntegrationClientModel
	if strings.Contains(integrationIDOrName, integrationNameSeparator) {
		parts := strings.SplitN(integrationIDOrName, integrationNameSeparator, 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid integration name: %s", integrationIDOrName)
		}

		integration = findIntegrationByTypeAndName(integrations, parts[0], parts[1])
	} else {
		integration = findIntegrationByID(integrations, integrationIDOrName)
	}

	if integration == nil {
		return nil, fmt.Errorf("integration %s not found", integrationIDOrName)
	}

	return integration, nil
}

func findIntegrationByTypeAndName(integrations []clientapi.IntegrationClientModel, integrationType string, integrationName string) *clientapi.IntegrationClientModel {
	for _, integration := range integrations {
		if integration.Type == integrationType && integration.Name == integrationName {
			return &integration
		}
	}

	return nil
}

func findIntegrationByID(integrations []clientapi.IntegrationClientModel, integrationID string) *clientapi.IntegrationClientModel {
	for _, integration := range integrations {
		if integration.Id == integrationID {
			return &integration
		}
	}

	return nil
}
