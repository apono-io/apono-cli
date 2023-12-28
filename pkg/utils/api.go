package utils

import (
	"context"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
)

func GetAllPages[T any](ctx context.Context, client *aponoapi.AponoClient, nextPageFunc func(context.Context, *aponoapi.AponoClient, int32) ([]T, *clientapi.PaginationClientInfoModel, error)) ([]T, error) {
	var result []T

	skip := 0
	hasNextPage := true
	for ok := true; ok; ok = hasNextPage {
		resp, pagination, err := nextPageFunc(ctx, client, int32(skip))
		if err != nil {
			return nil, err
		}

		result = append(result, resp...)

		hasNextPage = int(pagination.Limit) <= len(resp)
		skip += int(pagination.Limit)
	}

	return result, nil
}
