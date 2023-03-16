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
		clientID    string
		apiURL      string
		appURL      string
	)

	cmd := &cobra.Command{
		Use:     "login",
		GroupID: Group.ID,
		Short:   "Login to Apono",
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
						AuthURL:   fmt.Sprintf("%s/oauth/authorize", appURL),
						TokenURL:  fmt.Sprintf("%s/oauth/token", appURL),
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
				Logf:                   log.Printf,
			}

			eg, ctx := errgroup.WithContext(cmd.Context())
			eg.Go(func() error {
				select {
				case url := <-ready:
					log.Printf("Open %s", url)
					if err := browser.OpenURL(url); err != nil {
						log.Printf("could not open the browser: %s", err)
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

				log.Printf("You got a valid token until %s", token.Expiry)
				return storeProfileToken(profileName, clientID, apiURL, appURL, token)
			})
			if err := eg.Wait(); err != nil {
				log.Fatalf("authorization error: %s", err)
			}

			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&profileName, "profile", "p", "default", "profile name")
	flags.StringVarP(&clientID, clientIDFlagName, "", "3afae9ff-48e6-45f3-b0e8-37658b7271b7", "oauth client id")
	flags.StringVarP(&apiURL, apiURLFlagName, "", "https://api.apono.io", "apono api url")
	flags.StringVarP(&appURL, appURLFlagName, "", "https://app.apono.io", "apono app url")
	_ = flags.MarkHidden(clientIDFlagName)
	_ = flags.MarkHidden(apiURLFlagName)
	_ = flags.MarkHidden(appURLFlagName)
	return cmd
}

func storeProfileToken(profileName, clientID, apiURL, appURL string, token *oauth2.Token) error {
	cfg, err := config.Get()
	if err != nil {
		return err
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
		return fmt.Errorf("failed to parse access_token: %w", err)
	}

	if cfg.Auth.Profiles == nil {
		cfg.Auth.Profiles = make(map[config.ProfileName]config.SessionConfig)
	}

	cfg.Auth.Profiles[pn] = config.SessionConfig{
		ClientID:  clientID,
		ApiURL:    apiURL,
		AppURL:    appURL,
		AccountID: claims.AccountID,
		UserID:    claims.UserID,
		Token:     *token,
		CreatedAt: time.Now(),
	}

	return config.Save(cfg)
}
