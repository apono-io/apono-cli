package commands

import (
	"fmt"
	"log"

	"github.com/int128/oauth2cli"
	"github.com/int128/oauth2cli/oauth2params"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
)

var LoginCommand = &cobra.Command{
	Use:     "login",
	GroupID: Group.ID,
	Short:   "Login to Apono",
	Long:    "Login to Apono",
	RunE: func(cmd *cobra.Command, args []string) error {
		pkce, err := oauth2params.NewPKCE()
		if err != nil {
			return fmt.Errorf("failed to create code challenge: %w", err)
		}

		ready := make(chan string, 1)
		defer close(ready)
		cfg := oauth2cli.Config{
			OAuth2Config: oauth2.Config{
				ClientID: "apono-cli",
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
			return nil
		})
		if err := eg.Wait(); err != nil {
			log.Fatalf("authorization error: %s", err)
		}

		return nil
	},
}
