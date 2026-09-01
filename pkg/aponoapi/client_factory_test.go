package aponoapi

import (
	"testing"

	"github.com/apono-io/apono-cli/pkg/config"
)

func TestGetSessionConfig(t *testing.T) {
	t.Run("uses the env token when no profile is specified", func(t *testing.T) {
		t.Setenv(config.PersonalTokenEnvVar, "test-personal-token")

		sessionCfg, err := getSessionConfig("")
		if err != nil {
			t.Fatalf("getSessionConfig() error = %v, want nil", err)
		}
		if sessionCfg.PersonalToken != "test-personal-token" {
			t.Errorf("PersonalToken = %q, want %q", sessionCfg.PersonalToken, "test-personal-token")
		}
		if sessionCfg.ApiURL != config.APIDefaultURL {
			t.Errorf("ApiURL = %q, want %q", sessionCfg.ApiURL, config.APIDefaultURL)
		}
	})

	t.Run("an explicit profile takes precedence over the env token", func(t *testing.T) {
		t.Setenv(config.PersonalTokenEnvVar, "test-personal-token")

		// The profile lookup must be attempted (and fail for this nonexistent
		// profile) instead of falling back to the environment token.
		sessionCfg, err := getSessionConfig("nonexistent-profile-for-test")
		if err == nil {
			t.Fatalf("getSessionConfig() = %+v, want error for nonexistent profile", sessionCfg)
		}
	})
}
