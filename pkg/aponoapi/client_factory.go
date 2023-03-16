package aponoapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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

	sessionCfg := cfg.Auth.Profiles[config.ProfileName(profileName)]
	oauthHTTPClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&sessionCfg.Token))
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
