package analytics

import (
	"context"
	"errors"
	"time"
)

type contextKey string

const (
	startTimeContextKey = contextKey("__apono_start_time")
	commandIDContextKey = contextKey("__apono_command_id")
)

var (
	ErrValueNotConfigured  = errors.New("value is not set in context")
	ErrIllegalContextValue = errors.New("illegal value is set in context")
)

func CreateStartTimeContext(ctx context.Context, time *time.Time) context.Context {
	return context.WithValue(ctx, startTimeContextKey, time)
}

func CreateCommandIDContext(ctx context.Context, commandID string) context.Context {
	return context.WithValue(ctx, commandIDContextKey, commandID)
}

func GetStartTime(ctx context.Context) (*time.Time, error) {
	versionContext := ctx.Value(startTimeContextKey)
	if versionContext == nil {
		return nil, ErrValueNotConfigured
	}

	if versionInfo, ok := versionContext.(*time.Time); ok {
		return versionInfo, nil
	}

	return nil, ErrIllegalContextValue
}

func GetCommandID(ctx context.Context) (string, error) {
	versionContext := ctx.Value(commandIDContextKey)
	if versionContext == nil {
		return "", ErrValueNotConfigured
	}

	if versionInfo, ok := versionContext.(string); ok {
		return versionInfo, nil
	}

	return "", ErrIllegalContextValue
}
