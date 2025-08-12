package actions

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/commands/auth/actions"
	"github.com/apono-io/apono-cli/pkg/config"
	"github.com/apono-io/apono-cli/pkg/groups"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

func MCP() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "mcp",
		Short:             "Run stdio MCP proxy server",
		GroupID:           groups.OtherCommandsGroup.ID,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error { return nil },
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("=== MCP Authentication Testing ===")

			// Get current active profile
			cfg, err := config.Get()
			if err != nil {
				return fmt.Errorf("failed to get config: %w", err)
			}

			if cfg.Auth.ActiveProfile == "" {
				return fmt.Errorf("no active profile configured")
			}

			activeProfile := cfg.Auth.ActiveProfile
			fmt.Printf("\n1. Using active profile: %s\n", activeProfile)

			// Get profile configuration
			profile, err := config.GetProfileByName(activeProfile)
			if err != nil {
				return fmt.Errorf("failed to get profile %s: %w", activeProfile, err)
			}

			// Authenticate MCP client if needed
			fmt.Println("\n2. Authenticating MCP client...")
			if err := authenticateMCPClient(cmd.Context(), profile); err != nil {
				return fmt.Errorf("failed to authenticate MCP client: %w", err)
			}
			fmt.Println("✓ MCP client authenticated successfully")

			mcpURL, err := url.Parse(profile.ApiURL)
			if err != nil {
				return fmt.Errorf("failed to parse API URL: %w", err)
			}
			mcpURL.Host = "localhost:8108"
			mcpURL.Path = "/api/client/v1/mcp"

			// Test 2: Call MCP endpoint with MCP auth (expect 500)
			fmt.Println("\n4. Testing MCP endpoint with MCP auth (expect 500)...")

			// Create MCP client directly using the generic factory
			cfg, err = config.Get()
			if err != nil {
				return fmt.Errorf("failed to get config: %w", err)
			}

			// Create MCP-specific OAuth config and token source
			mcpClientID := "d568d746-d972-43a6-a697-33388d19ea76"
			oauthConfig := oauth2.Config{
				ClientID: mcpClientID,
				Endpoint: oauth2.Endpoint{
					AuthURL:   config.GetOAuthAuthURL(profile.AppURL),
					TokenURL:  config.GetOAuthTokenURL(profile.AppURL),
					AuthStyle: oauth2.AuthStyleInParams,
				},
			}

			ts := aponoapi.NewRefreshableTokenSource(cmd.Context(), oauthConfig, &cfg.MCP.Token, func(t *oauth2.Token) error {
				return config.SaveMCPToken(*t)
			})

			mcpClient, err := aponoapi.CreateClientWithTokenSource(cmd.Context(), string(activeProfile), ts, "apono-mcp")
			if err != nil {
				return fmt.Errorf("failed to create MCP client: %w", err)
			}

			// Test with MCP auth
			mcpResp, err := testMCPEndpoint(cmd.Context(), mcpURL.String(), mcpClient.ClientAPI.GetConfig().HTTPClient)
			if err != nil {
				return fmt.Errorf("failed to test with MCP auth: %w", err)
			}

			if mcpResp.StatusCode == 500 {
				fmt.Println("✓ Expected 500 Internal Server Error received with MCP auth")
			} else {
				fmt.Printf("⚠ Unexpected status code with MCP auth: %d (expected 500)\n", mcpResp.StatusCode)
			}

			fmt.Println("\n=== MCP Testing Complete ===")
			return nil
		},
	}

	return cmd
}

func authenticateMCPClient(ctx context.Context, profile *config.SessionConfig) error {
	// Check if MCP token already exists and is valid
	cfg, err := config.Get()
	if err != nil {
		return err
	}

	if cfg.MCP.Token.AccessToken != "" && cfg.MCP.Token.Valid() {
		return nil // Token already exists and is valid
	}

	// Use the existing login implementation with MCP-specific parameters
	return performMCPLogin(ctx, profile)
}

func performMCPLogin(ctx context.Context, profile *config.SessionConfig) error {
	// Use the existing OAuth implementation with MCP-specific parameters
	mcpClientID := "d568d746-d972-43a6-a697-33388d19ea76"
	mcpScopes := []string{
		"end_user:mcp",
	}

	oauthConfig := actions.OAuthConfig{
		ClientID:  mcpClientID,
		ApiURL:    profile.ApiURL,
		AppURL:    profile.AppURL,
		PortalURL: profile.PortalURL,
		Scopes:    mcpScopes,
		Verbose:   false,
	}

	// Get the OAuth token using the helper function
	oauthToken, err := actions.PerformOAuthLogin(ctx, oauthConfig)
	if err != nil {
		return fmt.Errorf("MCP OAuth authentication failed: %w", err)
	}

	// Store the MCP token separately
	return storeMCPToken(oauthToken)
}

func storeMCPToken(oauthToken *oauth2.Token) error {
	cfg, err := config.Get()
	if err != nil {
		return err
	}

	// Parse JWT to get account and user info
	type aponoClaims struct {
		AuthorizationID string   `json:"authorization_id"`
		AccountID       string   `json:"account_id"`
		UserID          string   `json:"user_id"`
		ClientID        string   `json:"client_id"`
		Scopes          []string `json:"scopes"`
		jwt.RegisteredClaims
	}

	claims := new(aponoClaims)
	_, _, err = jwt.NewParser().ParseUnverified(oauthToken.AccessToken, claims)
	if err != nil {
		return fmt.Errorf("failed to parse access_token: %w", err)
	}

	// Store MCP token
	cfg.MCP.Token = *oauthToken
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save MCP token: %w", err)
	}

	fmt.Printf("MCP client authenticated for account %s as user %s\n", claims.AccountID, claims.UserID)
	return nil
}

func testMCPEndpoint(ctx context.Context, url string, httpClient *http.Client) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	fmt.Printf("  Response: %s (Status: %d)\n", url, resp.StatusCode)
	return resp, nil
}
