package utils

import (
	"context"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
)

func GetAllPages[T any](ctx context.Context, client *aponoapi.AponoClient, nextPageFunc func(context.Context, *aponoapi.AponoClient, int32) ([]T, *clientapi.PaginationClientInfoModel, error)) ([]T, error) {
	var result []T

	skip := 0
	for {
		resp, pagination, err := nextPageFunc(ctx, client, int32(skip))
		if err != nil {
			return nil, err
		}

		result = append(result, resp...)

		skip += int(pagination.Limit)

		hasNextPage := int(pagination.Limit) <= len(resp)
		if !hasNextPage {
			break
		}
	}

	return result, nil
}
