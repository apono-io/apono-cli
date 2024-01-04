package services

import (
	"context"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/utils"
)

func ListAccessSessions(ctx context.Context, client *aponoapi.AponoClient, integrationIds []string, bundleIds []string) ([]clientapi.AccessSessionClientModel, error) {
	return utils.GetAllPages(ctx, client, func(ctx context.Context, client *aponoapi.AponoClient, skip int32) ([]clientapi.AccessSessionClientModel, *clientapi.PaginationClientInfoModel, error) {
		listBundleRequest := client.ClientAPI.AccessSessionsAPI.ListAccessSessions(ctx).Skip(skip)
		if integrationIds != nil {
			listBundleRequest = listBundleRequest.IntegrationId(integrationIds)
		}
		if bundleIds != nil {
			listBundleRequest = listBundleRequest.BundleId(bundleIds)
		}

		resp, _, err := listBundleRequest.Execute()
		if err != nil {
			return nil, nil, err
		}

		return resp.Data, &resp.Pagination, nil
	})
}
