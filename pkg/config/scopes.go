package config

import (
	"github.com/golang-jwt/jwt/v5"
)

const (
	ScopeAssistant = "end_user_client:read"
)

type aponoClaims struct {
	Scopes []string `json:"scopes"`
	jwt.RegisteredClaims
}

// SessionHasScope checks if the session's OAuth token contains the specified scope.
// Returns true for personal token auth (scope validation not applicable).
// Returns false if the session is nil or token parsing fails.
func SessionHasScope(session *SessionConfig, scope string) bool {
	if session == nil {
		return false
	}

	if session.PersonalToken != "" {
		return true
	}

	accessToken := session.Token.AccessToken
	if accessToken == "" {
		return false
	}

	claims := new(aponoClaims)
	_, _, err := jwt.NewParser().ParseUnverified(accessToken, claims)
	if err != nil {
		return false
	}

	for _, s := range claims.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}
