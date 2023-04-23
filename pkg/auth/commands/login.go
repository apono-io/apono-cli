package commands

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/int128/oauth2cli"
	"github.com/int128/oauth2cli/oauth2params"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"

	"github.com/apono-io/apono-cli/pkg/config"
)

const (
	clientIDFlagName = "client-id"
	apiURLFlagName   = "api-url"
	appURLFlagName   = "app-url"
)

func Login() *cobra.Command {
	var (
		profileName string
		verbose     bool
		clientID    string
		apiURL      string
		appURL      string
	)

	cmd := &cobra.Command{
		Use:               "login",
		GroupID:           Group.ID,
		Short:             "Login to Apono",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error { return nil },
		RunE: func(cmd *cobra.Command, args []string) error {
			apiURL = strings.TrimLeft(apiURL, "/")
			appURL = strings.TrimLeft(appURL, "/")
			pkce, err := oauth2params.NewPKCE()
			if err != nil {
				return fmt.Errorf("failed to create code challenge: %w", err)
			}

			ready := make(chan string, 1)
			defer close(ready)
			cfg := oauth2cli.Config{
				OAuth2Config: oauth2.Config{
					ClientID: clientID,
					Endpoint: oauth2.Endpoint{
						AuthURL:   config.GetOAuthAuthURL(appURL),
						TokenURL:  config.GetOAuthTokenURL(appURL),
						AuthStyle: oauth2.AuthStyleInParams,
					},
					Scopes: []string{
						"integrations:read",
						"requests:read",
						"requests:write",
					},
				},
				AuthCodeOptions:        pkce.AuthCodeOptions(),
				TokenRequestOptions:    pkce.TokenRequestOptions(),
				LocalServerReadyChan:   ready,
				LocalServerBindAddress: []string{"localhost:64131", "localhost:64132", "localhost:64133", "localhost:64134"},
			}

			if verbose {
				cfg.Logf = log.Printf
			}

			eg, ctx := errgroup.WithContext(cmd.Context())
			eg.Go(func() error {
				select {
				case url := <-ready:
					_, _ = fmt.Println("You will be redirected to your web browser to complete the login process")
					_, _ = fmt.Println("If the page did not open automatically, open this URL manually:", url)
					if err := browser.OpenURL(url); err != nil && verbose {
						log.Println("Could not open the browser:", err)
					}

					return nil
				case <-ctx.Done():
					return fmt.Errorf("context done while waiting for authorization: %w", ctx.Err())
				}
			})
			eg.Go(func() error {
				token, err := oauth2cli.GetToken(ctx, cfg)
				if err != nil {
					return fmt.Errorf("could not get a token: %w", err)
				}

				session, err := storeProfileToken(profileName, clientID, apiURL, appURL, token)
				if err != nil {
					return fmt.Errorf("could not store access token: %w", err)
				}

				log.Println("You successfully logged in to account", session.AccountID, "as", session.UserID)
				return nil
			})
			if err := eg.Wait(); err != nil {
				log.Fatalf("authorization error: %s", err)
			}

			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&profileName, "profile", "p", "default", "profile name")
	flags.BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	flags.StringVarP(&clientID, clientIDFlagName, "", "3afae9ff-48e6-45f3-b0e8-37658b7271b7", "oauth client id")
	flags.StringVarP(&apiURL, apiURLFlagName, "", "https://api.apono.io", "apono api url")
	flags.StringVarP(&appURL, appURLFlagName, "", "https://app.apono.io", "apono app url")
	_ = flags.MarkHidden(clientIDFlagName)
	_ = flags.MarkHidden(apiURLFlagName)
	_ = flags.MarkHidden(appURLFlagName)
	return cmd
}

func storeProfileToken(profileName, clientID, apiURL, appURL string, token *oauth2.Token) (*config.SessionConfig, error) {
	cfg, err := config.Get()
	if err != nil {
		return nil, err
	}

	pn := config.ProfileName(profileName)
	if cfg.Auth.ActiveProfile == "" {
		cfg.Auth.ActiveProfile = pn
	}

	type aponoClaims struct {
		AuthorizationID string   `json:"authorization_id"`
		AccountID       string   `json:"account_id"`
		UserID          string   `json:"user_id"`
		ClientID        string   `json:"client_id"`
		Scopes          []string `json:"scopes"`
		jwt.RegisteredClaims
	}

	claims := new(aponoClaims)
	_, _, err = jwt.NewParser().ParseUnverified(token.AccessToken, claims)
	if err != nil {
		return nil, fmt.Errorf("failed to parse access_token: %w", err)
	}

	if cfg.Auth.Profiles == nil {
		cfg.Auth.Profiles = make(map[config.ProfileName]config.SessionConfig)
	}

	session := config.SessionConfig{
		ClientID:  clientID,
		ApiURL:    apiURL,
		AppURL:    appURL,
		AccountID: claims.AccountID,
		UserID:    claims.UserID,
		Token:     *token,
		CreatedAt: time.Now(),
	}
	cfg.Auth.Profiles[pn] = session

	err = config.Save(cfg)
	if err != nil {
		return nil, err
	}

	return &session, nil
}
