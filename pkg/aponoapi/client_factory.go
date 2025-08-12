package aponoapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/apono-io/apono-sdk-go"
	"golang.org/x/oauth2"

	"github.com/apono-io/apono-cli/pkg/build"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/config"
)

const (
	authorizationHeaderKey = "Authorization"
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
	sessionCfg, err := config.GetProfileByName(config.ProfileName(profileName))
	if err != nil {
		return nil, err
	}

	oauthToken := &sessionCfg.Token
	personalToken := sessionCfg.PersonalToken
	var httpClient *http.Client

	if personalToken == "" {
		ts := NewRefreshableTokenSource(ctx, sessionCfg.GetOAuth2Config(), oauthToken, func(t *oauth2.Token) error {
			return saveOAuthToken(profileName, t)
		})
		httpClient = oauth2.NewClient(ctx, ts)
	} else {
		httpClient = HTTPClientWithPersonalToken(personalToken)
	}

	endpointURL, err := url.Parse(sessionCfg.ApiURL)
	if err != nil {
		return nil, fmt.Errorf("failed parsing url %s with error: %w", sessionCfg.ApiURL, err)
	}

	userAgent := fmt.Sprintf("apono-cli/%s (%s; %s)", build.Version, build.Commit, build.Date)
	return createClientWithConfig(endpointURL, httpClient, userAgent, sessionCfg)
}

func CreateClientWithTokenSource(ctx context.Context, profileName string, tokenSource oauth2.TokenSource, userAgent string) (*AponoClient, error) {
	sessionCfg, err := config.GetProfileByName(config.ProfileName(profileName))
	if err != nil {
		return nil, err
	}

	httpClient := oauth2.NewClient(ctx, tokenSource)

	endpointURL, err := url.Parse(sessionCfg.ApiURL)
	if err != nil {
		return nil, fmt.Errorf("failed parsing url %s with error: %w", sessionCfg.ApiURL, err)
	}

	return createClientWithConfig(endpointURL, httpClient, userAgent, sessionCfg)
}

func createClientWithConfig(endpointURL *url.URL, httpClient *http.Client, userAgent string, sessionCfg *config.SessionConfig) (*AponoClient, error) {
	adminAPIClientCfg := apono.NewConfiguration()
	adminAPIClientCfg.Scheme = endpointURL.Scheme
	adminAPIClientCfg.Host = endpointURL.Host
	adminAPIClientCfg.UserAgent = userAgent
	adminAPIClientCfg.HTTPClient = httpClient

	client := apono.NewAPIClient(adminAPIClientCfg)
	clientAPI := CreateClientAPI(endpointURL, httpClient, userAgent)

	return &AponoClient{
		APIClient: client,
		ClientAPI: clientAPI,
		Session: &Session{
			AccountID: sessionCfg.AccountID,
			UserID:    sessionCfg.UserID,
		},
	}, nil
}

func CreateClientAPI(endpointURL *url.URL, httpClient *http.Client, userAgent string) *clientapi.APIClient {
	clientAPIClientCfg := clientapi.NewConfiguration()
	clientAPIClientCfg.Scheme = endpointURL.Scheme
	clientAPIClientCfg.Host = endpointURL.Host
	clientAPIClientCfg.UserAgent = userAgent
	clientAPIClientCfg.HTTPClient = httpClient
	clientAPI := clientapi.NewAPIClient(clientAPIClientCfg)
	return clientAPI
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

func HTTPClientWithPersonalToken(personalToken string) *http.Client {
	client := &http.Client{
		Transport: &CustomHeaderTransport{
			Transport:   http.DefaultTransport,
			HeaderKey:   authorizationHeaderKey,
			HeaderValue: fmt.Sprintf("Bearer %s", personalToken),
		},
	}

	return client
}

type CustomHeaderTransport struct {
	Transport   http.RoundTripper
	HeaderKey   string
	HeaderValue string
}

func (t *CustomHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set(t.HeaderKey, t.HeaderValue)
	return t.Transport.RoundTrip(req)
}
