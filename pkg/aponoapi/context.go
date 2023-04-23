package aponoapi

import (
	"context"
	"errors"
)

type clientContextKey string

const contextKey = clientContextKey("__apono_client")

var (
	ErrClientNotConfigured = errors.New("client is not set in context")
	ErrIllegalContextValue = errors.New("illegal value is set in context")
)

func CreateContext(ctx context.Context, client *AponoClient) context.Context {
	return context.WithValue(ctx, contextKey, client)
}

func GetClient(ctx context.Context) (*AponoClient, error) {
	client := ctx.Value(contextKey)
	if client == nil {
		return nil, ErrClientNotConfigured
	}

	if aponoClient, ok := client.(*AponoClient); ok {
		return aponoClient, nil
	}

	return nil, ErrIllegalContextValue
}
