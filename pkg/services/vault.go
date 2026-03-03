package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	vclient "github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
)

const (
	VaultSecretSessionType     = "session-apono-vault-secret"
	VaultManagementSessionType = "session-apono-vault-management"
	AponoVaultIntegrationType  = "apono-vault"
)

// VaultCredentials holds cached vault connection credentials.
type VaultCredentials struct {
	VaultAddress string `json:"vault_address"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	MountName    string `json:"mount_name,omitempty"`
}

// VaultClient wraps the HashiCorp Vault SDK client for KV v2 operations.
type VaultClient struct {
	api *vclient.Client
}

// defaultCacheDir returns the default cache directory for vault credentials.
func defaultCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".apono", "cache")
	}

	return filepath.Join(home, ".apono", "cache")
}

// saveVaultCredentials saves vault credentials to a cache file as base64-encoded JSON.
func saveVaultCredentials(cacheDir string, integrationID string, creds *VaultCredentials) error {
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	filePath := filepath.Join(cacheDir, "vault-"+integrationID)

	if err := os.WriteFile(filePath, []byte(encoded), 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// loadVaultCredentials loads vault credentials from a cache file.
func loadVaultCredentials(cacheDir string, integrationID string) (*VaultCredentials, error) {
	filePath := filepath.Join(cacheDir, "vault-"+integrationID)

	encoded, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	data, err := base64.StdEncoding.DecodeString(string(encoded))
	if err != nil {
		return nil, fmt.Errorf("failed to decode cache file: %w", err)
	}

	var creds VaultCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	return &creds, nil
}

// ParseVaultPath splits a vault path like "kv/db/prod" into mount ("kv") and secretPath ("db/prod").
func ParseVaultPath(path string) (mount string, secretPath string, err error) {
	idx := strings.IndexByte(path, '/')
	if idx < 0 {
		return "", "", fmt.Errorf("invalid vault path %q: must contain at least one '/'", path)
	}

	mount = path[:idx]
	secretPath = path[idx+1:]

	if secretPath == "" {
		return "", "", fmt.Errorf("invalid vault path %q: secret path cannot be empty", path)
	}

	return mount, secretPath, nil
}

// FindVaultSession finds an active session matching the given integration ID and session type.
// Returns nil, nil if no matching session is found.
func FindVaultSession(ctx context.Context, client *aponoapi.AponoClient, integrationID string, sessionType string) (*clientapi.AccessSessionClientModel, error) {
	sessions, err := ListAccessSessions(ctx, client, []string{integrationID}, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list access sessions: %w", err)
	}

	for _, session := range sessions {
		if session.Type.Id == sessionType {
			return &session, nil
		}
	}

	return nil, nil
}

// ResolveVaultCredentials tries to load credentials from cache, falling back to
// the access details API. On successful API fetch, credentials are cached.
func ResolveVaultCredentials(ctx context.Context, client *aponoapi.AponoClient, integrationID string, session *clientapi.AccessSessionClientModel) (*VaultCredentials, error) {
	cacheDir := defaultCacheDir()

	cached, err := loadVaultCredentials(cacheDir, integrationID)
	if err == nil {
		return cached, nil
	}

	accessDetails, _, err := client.ClientAPI.AccessSessionsAPI.GetAccessSessionAccessDetails(ctx, session.Id).
		FormatType(JSONOutputFormat).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get access details for session %s: %w", session.Id, err)
	}

	jsonData := accessDetails.GetJson()

	vaultAddress, _ := jsonData["vault_address"].(string)
	username, _ := jsonData["username"].(string)
	password, _ := jsonData["password"].(string)
	mountName, _ := jsonData["mount_name"].(string)

	if vaultAddress == "" || username == "" || password == "" {
		return nil, fmt.Errorf("access details for session %s missing required vault credentials", session.Id)
	}

	creds := &VaultCredentials{
		VaultAddress: vaultAddress,
		Username:     username,
		Password:     password,
		MountName:    mountName,
	}

	// Best-effort cache; ignore errors.
	_ = saveVaultCredentials(cacheDir, integrationID, creds)

	return creds, nil
}

// ResolveVaultClient is a convenience function that finds a vault session, resolves
// credentials, and logs in to the vault. It returns the authenticated VaultClient
// and the resolved credentials (which include mount_name).
func ResolveVaultClient(ctx context.Context, client *aponoapi.AponoClient, vaultID string, sessionType string) (*VaultClient, *VaultCredentials, error) {
	integration, err := GetIntegrationByIDOrByTypeAndName(ctx, client, vaultID)
	if err != nil {
		return nil, nil, fmt.Errorf("vault %q not found", vaultID)
	}

	session, err := FindVaultSession(ctx, client, integration.Id, sessionType)
	if err != nil {
		return nil, nil, err
	}

	if session == nil {
		sessionLabel := "access"
		if sessionType == VaultManagementSessionType {
			sessionLabel = "management access"
		}

		return nil, nil, fmt.Errorf("no active %s found for vault %q, create a new request by running: apono request create", sessionLabel, vaultID)
	}

	creds, err := ResolveVaultCredentials(ctx, client, integration.Id, session)
	if err != nil {
		return nil, nil, err
	}

	vc, err := VaultLogin(ctx, creds.VaultAddress, creds.Username, creds.Password)
	if err != nil {
		return nil, nil, err
	}

	return vc, creds, nil
}

// VaultLogin authenticates to Vault using the userpass auth method and returns
// a VaultClient with the resulting client token.
func VaultLogin(ctx context.Context, address, username, password string) (*VaultClient, error) {
	address = strings.TrimRight(address, "/")

	api, err := vclient.New(
		vclient.WithAddress(address),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	resp, err := api.Auth.UserpassLogin(ctx, username, schema.UserpassLoginRequest{
		Password: password,
	})
	if err != nil {
		return nil, fmt.Errorf("vault login failed: %w", err)
	}

	if resp == nil || resp.Auth == nil || resp.Auth.ClientToken == "" {
		return nil, fmt.Errorf("vault login response missing client_token")
	}

	if err := api.SetToken(resp.Auth.ClientToken); err != nil {
		return nil, fmt.Errorf("failed to set vault token: %w", err)
	}

	return &VaultClient{api: api}, nil
}

// ReadSecret reads a KV v2 secret from Vault at the given mount and path.
func (vc *VaultClient) ReadSecret(ctx context.Context, mount, secretPath string) (map[string]interface{}, error) {
	resp, err := vc.api.Secrets.KvV2Read(ctx, secretPath, vclient.WithMountPath(mount))
	if err != nil {
		return nil, fmt.Errorf("vault read failed: %w", err)
	}

	if resp == nil || resp.Data.Data == nil {
		return nil, fmt.Errorf("vault read returned empty response")
	}

	return resp.Data.Data, nil
}

// WriteSecret writes a KV v2 secret to Vault at the given mount and path.
func (vc *VaultClient) WriteSecret(ctx context.Context, mount, secretPath string, data map[string]interface{}) error {
	_, err := vc.api.Secrets.KvV2Write(ctx, secretPath, schema.KvV2WriteRequest{
		Data: data,
	}, vclient.WithMountPath(mount))
	if err != nil {
		return fmt.Errorf("vault write failed: %w", err)
	}

	return nil
}

// ListSecrets lists KV v2 secret keys at the given mount and prefix.
func (vc *VaultClient) ListSecrets(ctx context.Context, mount string) ([]string, error) {
	resp, err := vc.api.Secrets.KvV2List(ctx, "", vclient.WithMountPath(mount))
	if err != nil {
		return nil, fmt.Errorf("vault list failed: %w", err)
	}

	if resp == nil {
		return nil, nil
	}

	return resp.Data.Keys, nil
}

// DeleteSecret deletes a KV v2 secret at the given mount and path.
func (vc *VaultClient) DeleteSecret(ctx context.Context, mount, secretPath string) error {
	_, err := vc.api.Secrets.KvV2Delete(ctx, secretPath, vclient.WithMountPath(mount))
	if err != nil {
		return fmt.Errorf("vault delete failed: %w", err)
	}

	return nil
}

// IsNotFoundError checks whether an error from the Vault SDK is a 404.
func IsNotFoundError(err error) bool {
	var responseError *vclient.ResponseError
	return errors.As(err, &responseError) && responseError.StatusCode == http.StatusNotFound
}
