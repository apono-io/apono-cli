package actions

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/int128/oauth2cli"
	"github.com/int128/oauth2cli/oauth2params"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"

	"github.com/apono-io/apono-cli/pkg/config"
)

type OAuthConfig struct {
	ClientID  string
	ApiURL    string
	AppURL    string
	PortalURL string
	TokenURL  string
	Scopes    []string
	Verbose   bool
}

func PerformOAuthLogin(ctx context.Context, oauthConfig OAuthConfig) (*oauth2.Token, error) {
	appURL := strings.TrimLeft(oauthConfig.AppURL, "/")
	tokenURL := strings.TrimLeft(oauthConfig.TokenURL, "/")

	pkce, err := oauth2params.NewPKCE()
	if err != nil {
		return nil, fmt.Errorf("failed to create code challenge: %w", err)
	}

	var oauthTokenURL string
	if tokenURL != "" {
		oauthTokenURL = tokenURL
	} else {
		oauthTokenURL = appURL
	}

	ready := make(chan string, 1)
	defer close(ready)

	cliConfig := oauth2cli.Config{
		OAuth2Config: oauth2.Config{
			ClientID: oauthConfig.ClientID,
			Endpoint: oauth2.Endpoint{
				AuthURL:   config.GetOAuthAuthURL(appURL),
				TokenURL:  config.GetOAuthTokenURL(oauthTokenURL),
				AuthStyle: oauth2.AuthStyleInParams,
			},
			Scopes: oauthConfig.Scopes,
		},
		AuthCodeOptions:        pkce.AuthCodeOptions(),
		TokenRequestOptions:    pkce.TokenRequestOptions(),
		LocalServerReadyChan:   ready,
		LocalServerBindAddress: []string{"localhost:64131", "localhost:64132", "localhost:64133", "localhost:64134"},
	}

	if oauthConfig.Verbose {
		cliConfig.Logf = log.Printf
	}

	eg, ctx := errgroup.WithContext(ctx)

	tokenChan := make(chan *oauth2.Token, 1)
	defer close(tokenChan)

	eg.Go(func() error {
		return loginViaBrowser(ready, oauthConfig.Verbose, ctx)
	})

	eg.Go(func() error {
		oauthToken, err := oauth2cli.GetToken(ctx, cliConfig)
		if err != nil {
			return fmt.Errorf("could not get oauth token: %w", err)
		}
		tokenChan <- oauthToken
		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, fmt.Errorf("authorization error: %w", err)
	}

	return <-tokenChan, nil
}

func loginViaBrowser(ready <-chan string, verbose bool, ctx context.Context) error {
	select {
	case loginURL := <-ready:
		fmt.Println("You will be redirected to your web browser to complete the login process")
		fmt.Println("If the page did not open automatically, open this URL manually:", loginURL)
		if err := browser.OpenURL(loginURL); err != nil && verbose {
			log.Println("Could not open the browser:", err)
		}

		return nil
	case <-ctx.Done():
		return fmt.Errorf("context done while waiting for authorization: %w", ctx.Err())
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout waiting for authorization")
	}
}
