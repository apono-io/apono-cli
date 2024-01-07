package services

import (
	"context"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/utils"
)

func ListAccessSessions(ctx context.Context, client *aponoapi.AponoClient, integrationIds []string, bundleIds []string) ([]clientapi.AccessSessionClientModel, error) {
	return utils.GetAllPages(ctx, client, func(ctx context.Context, client *aponoapi.AponoClient, skip int32) ([]clientapi.AccessSessionClientModel, *clientapi.PaginationClientInfoModel, error) {
		listSessionsRequest := client.ClientAPI.AccessSessionsAPI.ListAccessSessions(ctx).Skip(skip)
		if integrationIds != nil {
			listSessionsRequest = listSessionsRequest.IntegrationId(integrationIds)
		}
		if bundleIds != nil {
			listSessionsRequest = listSessionsRequest.BundleId(bundleIds)
		}

		resp, _, err := listSessionsRequest.Execute()
		if err != nil {
			return nil, nil, err
		}

		return resp.Data, &resp.Pagination, nil
	})
}
