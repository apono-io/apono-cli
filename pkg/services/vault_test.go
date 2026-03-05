package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadVaultCredentials(t *testing.T) {
	cacheDir := t.TempDir()
	integrationID := "test-integration-123"

	creds := &VaultCredentials{
		VaultAddress: "https://vault.example.com:8200",
		Username:     "testuser",
		Password:     "testpass",
	}

	err := saveVaultCredentials(cacheDir, integrationID, creds)
	if err != nil {
		t.Fatalf("saveVaultCredentials failed: %v", err)
	}

	loaded, err := loadVaultCredentials(cacheDir, integrationID)
	if err != nil {
		t.Fatalf("loadVaultCredentials failed: %v", err)
	}

	if loaded.VaultAddress != creds.VaultAddress {
		t.Errorf("VaultAddress: got %q, want %q", loaded.VaultAddress, creds.VaultAddress)
	}

	if loaded.Username != creds.Username {
		t.Errorf("Username: got %q, want %q", loaded.Username, creds.Username)
	}

	if loaded.Password != creds.Password {
		t.Errorf("Password: got %q, want %q", loaded.Password, creds.Password)
	}
}

func TestSaveAndLoadVaultCredentials_WithMountName(t *testing.T) {
	cacheDir := t.TempDir()
	integrationID := "test-mount"

	creds := &VaultCredentials{
		VaultAddress: "https://vault.example.com:8200",
		Username:     "testuser",
		Password:     "testpass",
		MountName:    "secret",
	}

	err := saveVaultCredentials(cacheDir, integrationID, creds)
	if err != nil {
		t.Fatalf("saveVaultCredentials failed: %v", err)
	}

	loaded, err := loadVaultCredentials(cacheDir, integrationID)
	if err != nil {
		t.Fatalf("loadVaultCredentials failed: %v", err)
	}

	if loaded.MountName != creds.MountName {
		t.Errorf("MountName: got %q, want %q", loaded.MountName, creds.MountName)
	}
}

func TestLoadVaultCredentials_NotFound(t *testing.T) {
	cacheDir := t.TempDir()

	_, err := loadVaultCredentials(cacheDir, "nonexistent-integration")
	if err == nil {
		t.Fatal("expected error for missing cache file, got nil")
	}
}

func TestSaveVaultCredentials_CreatesDirectory(t *testing.T) {
	baseDir := t.TempDir()
	nestedDir := filepath.Join(baseDir, "nested", "deep", "cache")
	integrationID := "dir-test"

	creds := &VaultCredentials{
		VaultAddress: "https://vault.example.com:8200",
		Username:     "user",
		Password:     "pass",
	}

	err := saveVaultCredentials(nestedDir, integrationID, creds)
	if err != nil {
		t.Fatalf("saveVaultCredentials failed: %v", err)
	}

	filePath := filepath.Join(nestedDir, "vault-"+integrationID)
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("cache file not created: %v", err)
	}

	if info.IsDir() {
		t.Fatal("expected file, got directory")
	}

	// Verify file permissions.
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file permissions: got %o, want 0600", perm)
	}

	// Verify the directory was created.
	dirInfo, err := os.Stat(nestedDir)
	if err != nil {
		t.Fatalf("cache directory not created: %v", err)
	}

	if !dirInfo.IsDir() {
		t.Fatal("expected directory")
	}
}

func TestParseVaultPath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantMount  string
		wantSecret string
	}{
		{
			name:       "standard path",
			path:       "kv/db/prod",
			wantMount:  "kv",
			wantSecret: "db/prod",
		},
		{
			name:       "simple path",
			path:       "kv/simple",
			wantMount:  "kv",
			wantSecret: "simple",
		},
		{
			name:       "deeply nested path",
			path:       "secret/nested/deep/path",
			wantMount:  "secret",
			wantSecret: "nested/deep/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mount, secretPath, err := ParseVaultPath(tt.path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if mount != tt.wantMount {
				t.Errorf("mount: got %q, want %q", mount, tt.wantMount)
			}

			if secretPath != tt.wantSecret {
				t.Errorf("secretPath: got %q, want %q", secretPath, tt.wantSecret)
			}
		})
	}
}

func TestParseVaultPath_Invalid(t *testing.T) {
	_, _, err := ParseVaultPath("noseparator")
	if err == nil {
		t.Fatal("expected error for path without separator, got nil")
	}
}

func TestParseVaultPath_EmptySecretPath(t *testing.T) {
	_, _, err := ParseVaultPath("kv/")
	if err == nil {
		t.Fatal("expected error for empty secret path, got nil")
	}
}
