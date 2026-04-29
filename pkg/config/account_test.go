package config

import (
	"errors"
	"path/filepath"
	"testing"
)

func withTempConfig(t *testing.T, cfg *Config) {
	t.Helper()
	origDir := DirPath
	origPath := configFilePath
	tmp := t.TempDir()
	DirPath = tmp
	configFilePath = filepath.Join(tmp, "config.json")
	t.Cleanup(func() {
		DirPath = origDir
		configFilePath = origPath
	})
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
}

func TestGetProfileByAccountID_match(t *testing.T) {
	withTempConfig(t, &Config{
		Auth: AuthConfig{
			ActiveProfile: "prod",
			Profiles: map[ProfileName]SessionConfig{
				"prod":    {AccountID: "acct-prod", ApiURL: "https://api.apono.io"},
				"staging": {AccountID: "acct-staging", ApiURL: "https://staging.apono.io"},
			},
		},
	})

	name, sess, err := GetProfileByAccountID("acct-staging")
	if err != nil {
		t.Fatalf("expected match, got error: %v", err)
	}
	if name != "staging" {
		t.Errorf("expected profile name 'staging', got %q", name)
	}
	if sess.AccountID != "acct-staging" {
		t.Errorf("expected session account_id 'acct-staging', got %q", sess.AccountID)
	}
	if sess.ApiURL != "https://staging.apono.io" {
		t.Errorf("expected staging api_url, got %q", sess.ApiURL)
	}
}

func TestGetProfileByAccountID_unknownAccountReturnsProfileNotExists(t *testing.T) {
	withTempConfig(t, &Config{
		Auth: AuthConfig{
			ActiveProfile: "prod",
			Profiles: map[ProfileName]SessionConfig{
				"prod": {AccountID: "acct-prod"},
			},
		},
	})

	_, _, err := GetProfileByAccountID("acct-nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown account")
	}
	if !errors.Is(err, ErrProfileNotExists) {
		t.Errorf("expected wrapped ErrProfileNotExists, got %v", err)
	}
}

func TestGetProfileByAccountID_noProfilesReturnsNoProfiles(t *testing.T) {
	withTempConfig(t, &Config{
		Auth: AuthConfig{
			Profiles: map[ProfileName]SessionConfig{},
		},
	})

	_, _, err := GetProfileByAccountID("acct-anything")
	if !errors.Is(err, ErrNoProfiles) {
		t.Errorf("expected ErrNoProfiles, got %v", err)
	}
}

func TestGetProfileByAccountID_returnsCopyNotPointerToMap(t *testing.T) {
	withTempConfig(t, &Config{
		Auth: AuthConfig{
			Profiles: map[ProfileName]SessionConfig{
				"prod": {AccountID: "acct-prod", AccountName: "original"},
			},
		},
	})

	_, sess, err := GetProfileByAccountID("acct-prod")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	// Mutating the returned session must not corrupt the on-disk config.
	sess.AccountName = "mutated"

	cfg, err := Get()
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if cfg.Auth.Profiles["prod"].AccountName != "original" {
		t.Errorf("returned session shouldn't share storage with on-disk config; got AccountName=%q", cfg.Auth.Profiles["prod"].AccountName)
	}
}
