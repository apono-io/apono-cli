package aponoapi

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"

	"github.com/apono-io/apono-cli/pkg/config"
)

type AponoClient struct {
	*ClientWithResponses
	Session *Session
}

type Session struct {
	AccountID string
	UserID    string
}

func CreateClient(ctx context.Context, profileName string) (*AponoClient, error) {
	cfg, err := config.Get()
	if err != nil {
		return nil, err
	}

	session := cfg.Auth.Profiles[config.ProfileName(profileName)]
	client, err := NewClientWithResponses(
		session.AponoURL,
		WithHTTPClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(&session.Token))),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create apono client: %w", err)
	}

	return &AponoClient{
		ClientWithResponses: client,
		Session: &Session{
			AccountID: session.AccountID,
			UserID:    session.UserID,
		},
	}, nil
}
