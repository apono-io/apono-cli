package aponoapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"

	"github.com/apono-io/apono-cli/pkg/config"
)

var ErrProfileNotExists = errors.New("profile not exists")

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
	client, err := NewClientWithResponses(
		sessionCfg.ApiURL,
		WithHTTPClient(&aponoHTTPClient{client: oauthHTTPClient}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create apono client: %w", err)
	}

	return &AponoClient{
		ClientWithResponses: client,
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

type aponoHTTPClient struct {
	client *http.Client
}

func (a *aponoHTTPClient) Do(req *http.Request) (*http.Response, error) {
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		bodyBytes, err := io.ReadAll(resp.Body)
		defer func() { _ = resp.Body.Close() }()
		if err != nil {
			return nil, err
		}

		message := string(bodyBytes)
		messageResponse := new(MessageResponse)
		if jsonErr := json.Unmarshal(bodyBytes, messageResponse); jsonErr == nil {
			message = messageResponse.Message
		}

		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, message)
	}

	return resp, nil
}
