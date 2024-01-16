package aponoapi

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/apono-io/apono-sdk-go"
	"golang.org/x/oauth2"

	"github.com/apono-io/apono-cli/pkg/build"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/config"
)

var (
	ErrProfileNotExists  = errors.New("profile not exists")
	ErrNoProfiles        = errors.New("no profiles configured, run `apono login` to create a profile")
	ErrorNoActiveProfile = errors.New("no active profile configured, run `apono login` to create a profile")
)

type AponoClient struct {
	*apono.APIClient
	ClientAPI *clientapi.APIClient
	Session   *Session
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

	if pn == "" {
		return nil, ErrorNoActiveProfile
	}

	if len(authConfig.Profiles) == 0 {
		return nil, ErrNoProfiles
	}

	sessionCfg, exists := authConfig.Profiles[pn]
	if !exists {
		if pn == "default" {
			return nil, ErrNoProfiles
		}

		return nil, fmt.Errorf("%s %s", pn, ErrProfileNotExists)
	}

	token := &sessionCfg.Token
	ts := NewRefreshableTokenSource(ctx, sessionCfg.GetOAuth2Config(), token, func(t *oauth2.Token) error {
		return saveOAuthToken(profileName, t)
	})

	oauthHTTPClient := oauth2.NewClient(ctx, ts)

	endpointURL, err := url.Parse(sessionCfg.ApiURL)
	if err != nil {
		return nil, fmt.Errorf("failed parsing url %s with error: %w", sessionCfg.ApiURL, err)
	}

	adminAPIClientCfg := apono.NewConfiguration()
	adminAPIClientCfg.Scheme = endpointURL.Scheme
	adminAPIClientCfg.Host = endpointURL.Host
	adminAPIClientCfg.UserAgent = fmt.Sprintf("apono-cli/%s (%s; %s)", build.Version, build.Commit, build.Date)
	adminAPIClientCfg.HTTPClient = oauthHTTPClient

	client := apono.NewAPIClient(adminAPIClientCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create apono client: %w", err)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create clientapi client: %w", err)
	}

	clientAPIClientCfg := clientapi.NewConfiguration()
	clientAPIClientCfg.Scheme = endpointURL.Scheme
	clientAPIClientCfg.Host = endpointURL.Host
	clientAPIClientCfg.UserAgent = fmt.Sprintf("apono-cli/%s (%s; %s)", build.Version, build.Commit, build.Date)
	clientAPIClientCfg.HTTPClient = oauthHTTPClient
	clientAPI := clientapi.NewAPIClient(clientAPIClientCfg)

	return &AponoClient{
		APIClient: client,
		ClientAPI: clientAPI,
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
