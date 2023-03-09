package commands

import (
	"fmt"
	"log"
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

func Login() *cobra.Command {
	profileName := new(string)
	cmd := &cobra.Command{
		Use:     "login",
		GroupID: Group.ID,
		Short:   "Login to Apono",
		RunE: func(cmd *cobra.Command, args []string) error {
			pkce, err := oauth2params.NewPKCE()
			if err != nil {
				return fmt.Errorf("failed to create code challenge: %w", err)
			}

			ready := make(chan string, 1)
			defer close(ready)
			cfg := oauth2cli.Config{
				OAuth2Config: oauth2.Config{
					ClientID: "799bbf9e-5e85-4fd1-9071-4368c7abfb57",
					Endpoint: oauth2.Endpoint{
						AuthURL:  "http://localhost:9000/oauth/authorize",
						TokenURL: "http://localhost:9000/oauth/token",
					},
					Scopes: []string{
						"requests:new",
						"requests:read",
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
				return storeProfileToken(*profileName, token)
			})
			if err := eg.Wait(); err != nil {
				log.Fatalf("authorization error: %s", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(profileName, "profile", "p", "default", "Profile name")
	return cmd
}

func storeProfileToken(profileName string, token *oauth2.Token) error {
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
		AccountID: claims.AccountID,
		UserID:    claims.UserID,
		Token:     *token,
		CreatedAt: time.Now(),
	}

	return config.Save(cfg)
}
