package config

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/oauth2"
)

const (
	APIDefaultURL    = "https://api.apono.io"
	AppDefaultURL    = "https://app.apono.io"
	PortalDefaultURL = "https://portal.apono.io"
)

var (
	ErrProfileNotExists  = errors.New("profile not exists")
	ErrNoProfiles        = errors.New("no profiles configured, run `apono login` to create a profile")
	ErrorNoActiveProfile = errors.New("no active profile configured, run `apono login` to create a profile")
)

type Config struct {
	Auth AuthConfig `json:"auth"`
	MCP  MCPConfig  `json:"mcp"`
}

type AuthConfig struct {
	ActiveProfile ProfileName                   `json:"active_profile"`
	Profiles      map[ProfileName]SessionConfig `json:"profiles"`
}

type ProfileName string

type SessionConfig struct {
	ClientID      string       `json:"client_id"`
	ApiURL        string       `json:"api_url"`
	AppURL        string       `json:"app_url"`
	PortalURL     string       `json:"portal_url"`
	AccountID     string       `json:"account_id"`
	UserID        string       `json:"user_id"`
	Token         oauth2.Token `json:"token"`
	CreatedAt     time.Time    `json:"created_at"`
	PersonalToken string       `json:"personal_token"`
}

type MCPConfig struct {
	Token oauth2.Token `json:"token"`
}

func (c SessionConfig) GetOAuth2Config() oauth2.Config {
	return oauth2.Config{
		ClientID: c.ClientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:   GetOAuthTokenURL(c.AppURL),
			TokenURL:  GetOAuthTokenURL(c.AppURL),
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}
}

func GetOAuthAuthURL(appURL string) string {
	return fmt.Sprintf("%s/oauth/authorize", appURL)
}

func GetOAuthTokenURL(appURL string) string {
	return fmt.Sprintf("%s/oauth/token", appURL)
}

func GetProfileByName(profileName ProfileName) (*SessionConfig, error) {
	cfg, err := Get()
	if err != nil {
		return nil, err
	}

	authConfig := cfg.Auth
	if len(authConfig.Profiles) == 0 {
		return nil, ErrNoProfiles
	}

	var pn ProfileName
	if profileName != "" {
		pn = profileName
	} else {
		pn = authConfig.ActiveProfile
		if pn == "" {
			return nil, ErrorNoActiveProfile
		}
	}

	sessionCfg, exists := authConfig.Profiles[pn]
	if !exists {
		if pn == "default" {
			return nil, ErrNoProfiles
		}

		return nil, fmt.Errorf("%s %s", pn, ErrProfileNotExists)
	}

	return &sessionCfg, nil
}

func SaveMCPToken(token oauth2.Token) error {
	cfg, err := Get()
	if err != nil {
		return err
	}

	cfg.MCP.Token = token
	return Save(cfg)
}
