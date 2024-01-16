package version

import (
	"context"
	"errors"
)

type contextKey string

const (
	versionContextKey = contextKey("__apono_version")
)

var (
	ErrValueNotConfigured  = errors.New("value is not set in context")
	ErrIllegalContextValue = errors.New("illegal value is set in context")
)

func CreateVersionContext(ctx context.Context, version *VersionInfo) context.Context {
	return context.WithValue(ctx, versionContextKey, version)
}

func GetVersion(ctx context.Context) (*VersionInfo, error) {
	versionContext := ctx.Value(versionContextKey)
	if versionContext == nil {
		return nil, ErrValueNotConfigured
	}

	if versionInfo, ok := versionContext.(*VersionInfo); ok {
		return versionInfo, nil
	}

	return nil, ErrIllegalContextValue
}
