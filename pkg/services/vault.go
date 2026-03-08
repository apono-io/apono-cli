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

	vclient "github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
)

const (
	AponoVaultIntegrationType = "apono-vault"
	DefaultVaultMount         = "apono-store"

	cacheDirPermission  = 0o700
	cacheFilePermission = 0o600
)

type VaultCredentials struct {
	VaultAddress string `json:"vault_address"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	MountName    string `json:"mount_name,omitempty"`
}

type VaultClient struct {
	api *vclient.Client
}

func defaultCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".apono", "cache")
	}

	return filepath.Join(home, ".apono", "cache")
}

func saveVaultCredentials(cacheDir string, integrationID string, creds *VaultCredentials) error {
	if err := os.MkdirAll(cacheDir, cacheDirPermission); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	filePath := filepath.Join(cacheDir, "vault-"+integrationID)

	if err := os.WriteFile(filePath, []byte(encoded), cacheFilePermission); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

func loadVaultCredentials(cacheDir string, integrationID string) (*VaultCredentials, error) {
	filePath := filepath.Join(cacheDir, "vault-"+integrationID)

	encoded, err := os.ReadFile(filePath) //nolint:gosec // path is built from fixed cache dir
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

func FindVaultSession(ctx context.Context, client *aponoapi.AponoClient, integrationID string) (*clientapi.AccessSessionClientModel, error) {
	sessions, err := ListAccessSessions(ctx, client, []string{integrationID}, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list access sessions: %w", err)
	}

	if len(sessions) > 0 {
		return &sessions[0], nil
	}

	return nil, nil
}

func ResolveVaultCredentials(ctx context.Context, client *aponoapi.AponoClient, integrationID string, session *clientapi.AccessSessionClientModel) (*VaultCredentials, error) {
	cacheDir := defaultCacheDir()
	resetMsg := fmt.Sprintf("reset them by running: apono access reset-credentials %s", session.Id)

	if isSessionCredentialsNew(session) {
		return fetchAndCacheCredentials(ctx, client, cacheDir, integrationID, session)
	}

	cached, err := loadVaultCredentials(cacheDir, integrationID)
	if err != nil || isMaskedPassword(cached.Password) {
		return nil, fmt.Errorf("vault credentials have already been consumed and are not cached locally; %s", resetMsg)
	}

	_, loginErr := VaultLogin(ctx, cached.VaultAddress, cached.Username, cached.Password)
	if loginErr != nil {
		_ = os.Remove(filepath.Join(cacheDir, "vault-"+integrationID))
		return nil, fmt.Errorf("vault credentials have expired; %s", resetMsg)
	}

	return cached, nil
}

func fetchAndCacheCredentials(ctx context.Context, client *aponoapi.AponoClient, cacheDir, integrationID string, session *clientapi.AccessSessionClientModel) (*VaultCredentials, error) {
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

	if vaultAddress == "" || username == "" || password == "" || isMaskedPassword(password) {
		return nil, fmt.Errorf("vault credentials are not available for session %s; reset them by running: apono access reset-credentials %s", session.Id, session.Id)
	}

	creds := &VaultCredentials{
		VaultAddress: vaultAddress,
		Username:     username,
		Password:     password,
		MountName:    mountName,
	}

	_ = saveVaultCredentials(cacheDir, integrationID, creds)

	return creds, nil
}

func findVaultIntegration(ctx context.Context, client *aponoapi.AponoClient, nameOrID string) (*clientapi.IntegrationClientModel, error) {
	integrations, err := ListIntegrations(ctx, client)
	if err != nil {
		return nil, err
	}

	for _, integration := range integrations {
		if integration.Name == nameOrID || integration.Id == nameOrID {
			if integration.Type != AponoVaultIntegrationType {
				return nil, fmt.Errorf("integration %q is of type %q, not %q", nameOrID, integration.Type, AponoVaultIntegrationType)
			}

			return &integration, nil
		}
	}

	return nil, fmt.Errorf("vault %q not found", nameOrID)
}

func ResolveVaultClient(ctx context.Context, client *aponoapi.AponoClient, vaultID string) (*VaultClient, *VaultCredentials, error) {
	integration, err := findVaultIntegration(ctx, client, vaultID)
	if err != nil {
		return nil, nil, err
	}

	session, err := FindVaultSession(ctx, client, integration.Id)
	if err != nil {
		return nil, nil, err
	}

	if session == nil {
		return nil, nil, fmt.Errorf("no active session found for vault %q, create a new request by running: apono request create", vaultID)
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

func (vc *VaultClient) WriteSecret(ctx context.Context, mount, secretPath string, data map[string]interface{}) error {
	_, err := vc.api.Secrets.KvV2Write(ctx, secretPath, schema.KvV2WriteRequest{
		Data: data,
	}, vclient.WithMountPath(mount))
	if err != nil {
		return fmt.Errorf("vault write failed: %w", err)
	}

	return nil
}

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

func (vc *VaultClient) SecretExists(ctx context.Context, mount, secretPath string) (bool, error) {
	_, err := vc.api.Secrets.KvV2Read(ctx, secretPath, vclient.WithMountPath(mount))
	if err != nil {
		if IsNotFoundError(err) {
			return false, nil
		}

		return false, fmt.Errorf("vault read failed: %w", err)
	}

	return true, nil
}

func (vc *VaultClient) DeleteSecret(ctx context.Context, mount, secretPath string) error {
	_, err := vc.api.Secrets.KvV2Delete(ctx, secretPath, vclient.WithMountPath(mount))
	if err != nil {
		if IsNotFoundError(err) {
			return fmt.Errorf("secret %q not found in mount %q", secretPath, mount)
		}

		return fmt.Errorf("vault delete failed: %w", err)
	}

	return nil
}

func IsNotFoundError(err error) bool {
	var responseError *vclient.ResponseError
	return errors.As(err, &responseError) && responseError.StatusCode == http.StatusNotFound
}

func isSessionCredentialsNew(session *clientapi.AccessSessionClientModel) bool {
	if session.Credentials.IsSet() {
		return strings.EqualFold(session.Credentials.Get().Status, newCredentialsStatus)
	}

	return false
}

func isMaskedPassword(password string) bool {
	for _, r := range password {
		if r != '*' {
			return false
		}
	}

	return len(password) > 0
}
