package config

import (
	"fmt"
	"time"

	"golang.org/x/oauth2"
)

type Config struct {
	Auth AuthConfig `json:"auth"`
}

type AuthConfig struct {
	ActiveProfile ProfileName                   `json:"active_profile"`
	Profiles      map[ProfileName]SessionConfig `json:"profiles"`
}

type ProfileName string

type SessionConfig struct {
	ClientID  string       `json:"client_id"`
	ApiURL    string       `json:"api_url"`
	AppURL    string       `json:"app_url"`
	AccountID string       `json:"account_id"`
	UserID    string       `json:"user_id"`
	Token     oauth2.Token `json:"token"`
	CreatedAt time.Time    `json:"created_at"`
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
