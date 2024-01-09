package services

import (
	"context"
	"fmt"

	"github.com/apono-io/apono-cli/pkg/utils"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
)

func GetBundleByNameOrID(ctx context.Context, client *aponoapi.AponoClient, bundleNameOrID string) (*clientapi.BundleClientModel, error) {
	bundles, err := ListBundles(ctx, client, "")
	if err != nil {
		return nil, err
	}

	for _, bundle := range bundles {
		if bundle.Name == bundleNameOrID || bundle.Id == bundleNameOrID {
			return &bundle, nil
		}
	}

	return nil, fmt.Errorf("bundle %s not found", bundleNameOrID)
}

func ListBundles(ctx context.Context, client *aponoapi.AponoClient, search string) ([]clientapi.BundleClientModel, error) {
	return utils.GetAllPages(ctx, client, func(ctx context.Context, client *aponoapi.AponoClient, skip int32) ([]clientapi.BundleClientModel, *clientapi.PaginationClientInfoModel, error) {
		listBundleRequest := client.ClientAPI.InventoryAPI.ListAccessBundles(ctx).Skip(skip)
		if search != "" {
			listBundleRequest.Search(search)
		}

		resp, _, err := listBundleRequest.Execute()
		if err != nil {
			return nil, nil, err
		}

		return resp.Data, &resp.Pagination, nil
	})
}
