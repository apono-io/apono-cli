package utils

import (
	"context"
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
)

func GetBundleByNameOrId(ctx context.Context, client *aponoapi.AponoClient, bundleNameOrId string) (*clientapi.BundleClientModel, error) {
	bundles, err := ListBundles(ctx, client)
	if err != nil {
		return nil, err
	}

	for _, bundle := range bundles {
		if bundle.Name == bundleNameOrId || bundle.Id == bundleNameOrId {
			return &bundle, nil
		}
	}

	return nil, fmt.Errorf("bundle %s not found", bundleNameOrId)
}

func ListBundles(ctx context.Context, client *aponoapi.AponoClient) ([]clientapi.BundleClientModel, error) {
	return GetAllPages(ctx, client, func(ctx context.Context, client *aponoapi.AponoClient, skip int32) ([]clientapi.BundleClientModel, *clientapi.PaginationClientInfoModel, error) {
		resp, _, err := client.ClientAPI.InventoryAPI.ListAccessBundles(ctx).
			Skip(skip).
			Execute()
		if err != nil {
			return nil, nil, err
		}

		return resp.Data, &resp.Pagination, nil
	})
}
