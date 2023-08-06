package aponoapi

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/apono-io/apono-sdk-go"
	"golang.org/x/oauth2"

	"github.com/apono-io/apono-cli/pkg/config"
)

var ErrProfileNotExists = errors.New("profile not exists")

type AponoClient struct {
	*apono.APIClient
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

	authConfig := cfg.Auth
	pn := authConfig.ActiveProfile
	if profileName != "" {
		pn = config.ProfileName(profileName)
	}

	sessionCfg, exists := authConfig.Profiles[pn]
	if !exists {
		return nil, ErrProfileNotExists
	}

	token := &sessionCfg.Token
	ts := NewRefreshableTokenSource(ctx, sessionCfg.GetOAuth2Config(), token, func(t *oauth2.Token) error {
		return saveOAuthToken(profileName, t)
	})

	oauthHTTPClient := oauth2.NewClient(ctx, ts)

	endpointURL, err := url.Parse(sessionCfg.ApiURL)
	clientCfg := apono.NewConfiguration()
	clientCfg.Scheme = endpointURL.Scheme
	clientCfg.Host = endpointURL.Host
	clientCfg.UserAgent = fmt.Sprintf("apono-cli/%s", "p.version")
	clientCfg.HTTPClient = oauthHTTPClient

	client := apono.NewAPIClient(clientCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create apono client: %w", err)
	}

	return &AponoClient{
		APIClient: client,
		Session: &Session{
			AccountID: sessionCfg.AccountID,
			UserID:    sessionCfg.UserID,
		},
	}, nil
}

func saveOAuthToken(profileName string, t *oauth2.Token) error {
	cfg, err := config.Get()
	if err != nil {
		return err
	}

	sessionCfg := cfg.Auth.Profiles[config.ProfileName(profileName)]
	sessionCfg.Token = *t
	cfg.Auth.Profiles[config.ProfileName(profileName)] = sessionCfg
	return config.Save(cfg)
}
