package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
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

// VaultClient is an HTTP client for interacting with a HashiCorp Vault server.
type VaultClient struct {
	Address string
	Token   string
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

// VaultKVDataPath returns the KV v2 data API path for the given mount and secret path.
func VaultKVDataPath(mount, secretPath string) string {
	return mount + "/data/" + secretPath
}

// VaultKVMetadataPath returns the KV v2 metadata API path for the given mount.
func VaultKVMetadataPath(mount string) string {
	return mount + "/metadata/"
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

	vc, err := VaultLogin(creds.VaultAddress, creds.Username, creds.Password)
	if err != nil {
		return nil, nil, err
	}

	return vc, creds, nil
}

// VaultLogin authenticates to Vault using the userpass auth method and returns
// a VaultClient with the resulting client token.
func VaultLogin(address, username, password string) (*VaultClient, error) {
	address = strings.TrimRight(address, "/")
	loginPath := fmt.Sprintf("/v1/auth/userpass/login/%s", username)
	body, err := json.Marshal(map[string]string{"password": password})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal login request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, address+loginPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vault login request failed: %w", err)
	}

	defer resp.Body.Close()

	if err := checkVaultResponse(resp); err != nil {
		return nil, fmt.Errorf("vault login failed: %w", err)
	}

	var result struct {
		Auth struct {
			ClientToken string `json:"client_token"`
		} `json:"auth"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode vault login response: %w", err)
	}

	if result.Auth.ClientToken == "" {
		return nil, fmt.Errorf("vault login response missing client_token")
	}

	return &VaultClient{
		Address: address,
		Token:   result.Auth.ClientToken,
	}, nil
}

// Read reads a secret from Vault at the given API path.
func (vc *VaultClient) Read(apiPath string) (map[string]interface{}, error) {
	req, err := http.NewRequest(http.MethodGet, vc.Address+"/v1/"+apiPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create read request: %w", err)
	}

	req.Header.Set("X-Vault-Token", vc.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vault read request failed: %w", err)
	}

	defer resp.Body.Close()

	if err := checkVaultResponse(resp); err != nil {
		return nil, fmt.Errorf("vault read failed: %w", err)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode vault read response: %w", err)
	}

	return result, nil
}

// Write writes data to Vault at the given API path.
func (vc *VaultClient) Write(apiPath string, data map[string]interface{}) error {
	payload := map[string]interface{}{
		"data": data,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal write request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, vc.Address+"/v1/"+apiPath, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create write request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Vault-Token", vc.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("vault write request failed: %w", err)
	}

	defer resp.Body.Close()

	if err := checkVaultResponse(resp); err != nil {
		return fmt.Errorf("vault write failed: %w", err)
	}

	return nil
}

// List lists keys at the given metadata path in Vault using the LIST HTTP method.
func (vc *VaultClient) List(metadataPath string) ([]string, error) {
	req, err := http.NewRequest("LIST", vc.Address+"/v1/"+metadataPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list request: %w", err)
	}

	req.Header.Set("X-Vault-Token", vc.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vault list request failed: %w", err)
	}

	defer resp.Body.Close()

	if err := checkVaultResponse(resp); err != nil {
		return nil, fmt.Errorf("vault list failed: %w", err)
	}

	var result struct {
		Data struct {
			Keys []string `json:"keys"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode vault list response: %w", err)
	}

	return result.Data.Keys, nil
}

// Delete deletes a secret at the given API path in Vault.
func (vc *VaultClient) Delete(apiPath string) error {
	req, err := http.NewRequest(http.MethodDelete, vc.Address+"/v1/"+apiPath, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	req.Header.Set("X-Vault-Token", vc.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("vault delete request failed: %w", err)
	}

	defer resp.Body.Close()

	if err := checkVaultResponse(resp); err != nil {
		return fmt.Errorf("vault delete failed: %w", err)
	}

	return nil
}

// checkVaultResponse checks an HTTP response from Vault and returns an appropriate error.
func checkVaultResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("not found (404): %s", string(body))
	}

	return fmt.Errorf("operation failed with status %d: %s", resp.StatusCode, string(body))
}
