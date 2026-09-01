package config

import (
	"testing"
)

func TestGetSessionConfigFromEnv(t *testing.T) {
	t.Run("returns nil when the token env var is not set", func(t *testing.T) {
		t.Setenv(PersonalTokenEnvVar, "")

		if sessionCfg := GetSessionConfigFromEnv(); sessionCfg != nil {
			t.Errorf("GetSessionConfigFromEnv() = %+v, want nil", sessionCfg)
		}
	})

	t.Run("returns a personal token session with default URLs", func(t *testing.T) {
		t.Setenv(PersonalTokenEnvVar, "test-personal-token")
		t.Setenv(APIURLEnvVar, "")

		sessionCfg := GetSessionConfigFromEnv()
		if sessionCfg == nil {
			t.Fatal("GetSessionConfigFromEnv() = nil, want session config")
		}
		if sessionCfg.PersonalToken != "test-personal-token" {
			t.Errorf("PersonalToken = %q, want %q", sessionCfg.PersonalToken, "test-personal-token")
		}
		if sessionCfg.ApiURL != APIDefaultURL {
			t.Errorf("ApiURL = %q, want %q", sessionCfg.ApiURL, APIDefaultURL)
		}
		if sessionCfg.AppURL != AppDefaultURL {
			t.Errorf("AppURL = %q, want %q", sessionCfg.AppURL, AppDefaultURL)
		}
		if sessionCfg.PortalURL != PortalDefaultURL {
			t.Errorf("PortalURL = %q, want %q", sessionCfg.PortalURL, PortalDefaultURL)
		}
	})

	t.Run("honors the API URL override env var", func(t *testing.T) {
		t.Setenv(PersonalTokenEnvVar, "test-personal-token")
		t.Setenv(APIURLEnvVar, "https://api.example.apono.dev")

		sessionCfg := GetSessionConfigFromEnv()
		if sessionCfg == nil {
			t.Fatal("GetSessionConfigFromEnv() = nil, want session config")
		}
		if sessionCfg.ApiURL != "https://api.example.apono.dev" {
			t.Errorf("ApiURL = %q, want %q", sessionCfg.ApiURL, "https://api.example.apono.dev")
		}
	})
}
