package config

import (
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
	AccountID string       `json:"account_id"`
	UserID    string       `json:"user_id"`
	Token     oauth2.Token `json:"token"`
	CreatedAt time.Time    `json:"created_at"`
}
