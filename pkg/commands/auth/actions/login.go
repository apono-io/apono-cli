package actions

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/apono-io/apono-cli/pkg/build"
	"github.com/apono-io/apono-cli/pkg/groups"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/config"
)

const (
	clientIDFlagName      = "client-id"
	apiURLFlagName        = "api-url"
	appURLFlagName        = "app-url"
	portalURLFlagName     = "portal-url"
	tokenURLFlagName      = "token-url"
	personalTokenFlagName = "personal-token"
)

type loginCommandFlags struct {
	profileName   string
	verbose       bool
	clientID      string
	apiURL        string
	appURL        string
	portalURL     string
	tokenURL      string
	personalToken string
}

func Login() *cobra.Command {
	cmdFlags := loginCommandFlags{}

	cmd := &cobra.Command{
		Use:               "login",
		Short:             "Login to Apono",
		GroupID:           groups.AuthCommandsGroup.ID,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error { return nil },
		RunE: func(cmd *cobra.Command, args []string) error {
			apiURL := strings.TrimLeft(cmdFlags.apiURL, "/")
			appURL := strings.TrimLeft(cmdFlags.appURL, "/")
			portalURL := strings.TrimLeft(cmdFlags.portalURL, "/")
			tokenURL := strings.TrimLeft(cmdFlags.tokenURL, "/")
			personalToken := strings.TrimLeft(cmdFlags.personalToken, "")

			if personalToken != "" {
				return storeAndLogProfileToken(cmdFlags.profileName, cmdFlags.clientID, apiURL, appURL, portalURL, nil, personalToken, cmd.Context())
			}

			oauthConfig := OAuthConfig{
				ClientID:  cmdFlags.clientID,
				ApiURL:    apiURL,
				AppURL:    appURL,
				PortalURL: portalURL,
				TokenURL:  tokenURL,
				Scopes: []string{
					"end_user:access_sessions:read",
					"end_user:access_sessions:write",
					"end_user:access_requests:read",
					"end_user:access_requests:write",
					"end_user:inventory:read",
					"end_user:analytics:write",
				},
				Verbose: cmdFlags.verbose,
			}

			oauthToken, err := PerformOAuthLogin(cmd.Context(), oauthConfig)
			if err != nil {
				return fmt.Errorf("OAuth authentication failed: %w", err)
			}

			return storeAndLogProfileToken(cmdFlags.profileName, cmdFlags.clientID, apiURL, appURL, portalURL, oauthToken, "", cmd.Context())
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&cmdFlags.profileName, "profile", "p", "default", "profile name")
	flags.BoolVarP(&cmdFlags.verbose, "verbose", "v", false, "verbose output")
	flags.StringVarP(&cmdFlags.clientID, clientIDFlagName, "", "3afae9ff-48e6-45f3-b0e8-37658b7271b7", "oauth client id")
	flags.StringVarP(&cmdFlags.apiURL, apiURLFlagName, "", config.APIDefaultURL, "apono api url")
	flags.StringVarP(&cmdFlags.appURL, appURLFlagName, "", config.AppDefaultURL, "apono app url")
	flags.StringVarP(&cmdFlags.portalURL, portalURLFlagName, "", config.PortalDefaultURL, "apono portal url")
	flags.StringVarP(&cmdFlags.tokenURL, tokenURLFlagName, "", "", "apono token api url")
	flags.StringVarP(&cmdFlags.personalToken, personalTokenFlagName, "", "", "Log in to Apono with user personal token")

	_ = flags.MarkHidden(clientIDFlagName)
	_ = flags.MarkHidden(apiURLFlagName)
	_ = flags.MarkHidden(appURLFlagName)
	_ = flags.MarkHidden(portalURLFlagName)
	_ = flags.MarkHidden(tokenURLFlagName)
	return cmd
}

func storeAndLogProfileToken(profileName, clientID, apiURL, appURL, portalURL string, oauthToken *oauth2.Token, personalToken string, ctx context.Context) error {
	session, err := storeProfileToken(profileName, clientID, apiURL, appURL, portalURL, oauthToken, personalToken, ctx)
	if err != nil {
		return fmt.Errorf("could not store access oauthToken: %w", err)
	}

	fmt.Println("You successfully logged in to account", session.AccountID, "as", session.UserID)
	return nil
}

func storeProfileToken(profileName, clientID, apiURL, appURL, portalURL string, oauthToken *oauth2.Token, personalToken string, ctx context.Context) (*config.SessionConfig, error) {
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

	var accountID string
	var userID string

	if oauthToken != nil {
		claims := new(aponoClaims)
		_, _, err = jwt.NewParser().ParseUnverified(oauthToken.AccessToken, claims)
		if err != nil {
			return nil, fmt.Errorf("failed to parse access_token: %w", err)
		}
		accountID = claims.AccountID
		userID = claims.UserID
	} else {
		endpointURL, urlParseErr := url.Parse(apiURL)
		if urlParseErr != nil {
			return nil, fmt.Errorf("failed parsing url %s with error: %w", portalURL, urlParseErr)
		}
		userAgent := fmt.Sprintf("apono-cli/%s (%s; %s)", build.Version, build.Commit, build.Date)
		clientAPI := aponoapi.CreateClientAPI(endpointURL, aponoapi.HTTPClientWithPersonalToken(personalToken), userAgent)
		userSession, _, userSessionErr := clientAPI.UserSessionAPI.GetUserSession(ctx).Execute()
		if userSessionErr != nil {
			return nil, fmt.Errorf("failed fetching user session with error: %w", userSessionErr)
		}
		accountID = userSession.Account.Id
		userID = userSession.User.Id
	}

	if cfg.Auth.Profiles == nil {
		cfg.Auth.Profiles = make(map[config.ProfileName]config.SessionConfig)
	}

	session := config.SessionConfig{
		ClientID:      clientID,
		ApiURL:        apiURL,
		AppURL:        appURL,
		PortalURL:     portalURL,
		AccountID:     accountID,
		UserID:        userID,
		CreatedAt:     time.Now(),
		PersonalToken: personalToken,
	}
	if oauthToken != nil {
		session.Token = *oauthToken
	}
	cfg.Auth.Profiles[pn] = session

	err = config.Save(cfg)
	if err != nil {
		return nil, err
	}

	return &session, nil
}
