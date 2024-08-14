package config

import (
	"context"
	"errors"
)

type contextKey string

const (
	currentProfileContextKey = contextKey("__current_profile")
)

func CreateProfileContext(ctx context.Context, profileName string) context.Context {
	sessionCfg, err := GetProfileByName(ProfileName(profileName))
	if err != nil {
		return ctx
	}

	return context.WithValue(ctx, currentProfileContextKey, sessionCfg)
}

func GetCurrentProfile(ctx context.Context) (*SessionConfig, error) {
	sessionConfig := ctx.Value(currentProfileContextKey)
	if sessionConfig == nil {
		return nil, errors.New("no profile set in context")
	}

	if session, ok := sessionConfig.(*SessionConfig); ok {
		return session, nil
	}

	return nil, errors.New("illegal value set in context")
}
