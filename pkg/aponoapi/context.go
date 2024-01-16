package aponoapi

import (
	"context"
	"errors"
)

type contextKey string

const (
	clientContextKey = contextKey("__apono_client")
)

var (
	ErrClientNotConfigured = errors.New("client is not set in context")
	ErrIllegalContextValue = errors.New("illegal value is set in context")
)

func CreateClientContext(ctx context.Context, client *AponoClient) context.Context {
	return context.WithValue(ctx, clientContextKey, client)
}

func GetClient(ctx context.Context) (*AponoClient, error) {
	client := ctx.Value(clientContextKey)
	if client == nil {
		return nil, ErrClientNotConfigured
	}

	if aponoClient, ok := client.(*AponoClient); ok {
		return aponoClient, nil
	}

	return nil, ErrIllegalContextValue
}
