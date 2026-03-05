# Vault CLI Commands Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `apono vault` command group for managing and viewing secrets in Apono's native vault (DVL-8148, DVL-8149).

**Architecture:** New `vault` command group follows existing Configurator pattern. A services layer handles session discovery, credential caching (`~/.apono/cache/vault-<integration-id>`), and direct HTTP calls to vault (no SDK). The `--vault-id` flag accepts both integration ID and name.

**Tech Stack:** Go 1.24, Cobra CLI, Apono client API (generated), plain `net/http` for vault operations.

**Design Doc:** `docs/plans/2026-03-03-vault-commands-design.md`

---

### Task 1: Vault Service — Credential Caching

Core logic that all commands depend on: resolve vault-id, find active session, cache/retrieve credentials, authenticate to vault.

**Files:**
- Create: `pkg/services/vault.go`
- Test: `pkg/services/vault_test.go`

**Step 1: Write tests for credential cache read/write**

```go
// pkg/services/vault_test.go
package services

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadVaultCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, ".apono", "cache")

	creds := &VaultCredentials{
		VaultAddress: "https://vault.example.com",
		Username:     "test_apono",
		Password:     "s3cr3t",
	}

	err := saveVaultCredentials(cacheDir, "integration-123", creds)
	if err != nil {
		t.Fatalf("saveVaultCredentials failed: %v", err)
	}

	loaded, err := loadVaultCredentials(cacheDir, "integration-123")
	if err != nil {
		t.Fatalf("loadVaultCredentials failed: %v", err)
	}

	if loaded.VaultAddress != creds.VaultAddress {
		t.Errorf("VaultAddress = %q, want %q", loaded.VaultAddress, creds.VaultAddress)
	}
	if loaded.Username != creds.Username {
		t.Errorf("Username = %q, want %q", loaded.Username, creds.Username)
	}
	if loaded.Password != creds.Password {
		t.Errorf("Password = %q, want %q", loaded.Password, creds.Password)
	}
}

func TestLoadVaultCredentials_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := loadVaultCredentials(tmpDir, "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing cache, got nil")
	}
}

func TestSaveVaultCredentials_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "deep", "nested", "cache")

	creds := &VaultCredentials{
		VaultAddress: "https://vault.example.com",
		Username:     "user",
		Password:     "pass",
	}

	err := saveVaultCredentials(cacheDir, "int-456", creds)
	if err != nil {
		t.Fatalf("saveVaultCredentials failed: %v", err)
	}

	cachePath := filepath.Join(cacheDir, "vault-int-456")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatal("cache file was not created")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./pkg/services/ -run TestSaveAndLoadVaultCredentials -v`
Expected: FAIL — `VaultCredentials` type and functions undefined.

**Step 3: Implement credential caching**

```go
// pkg/services/vault.go
package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	vaultCacheFilePrefix     = "vault-"
	aponoVaultIntegrationType = "apono-vault"
)

// VaultCredentials holds cached vault authentication credentials.
type VaultCredentials struct {
	VaultAddress string `json:"vault_address"`
	Username     string `json:"username"`
	Password     string `json:"password"`
}

func defaultCacheDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".apono", "cache")
}

func saveVaultCredentials(cacheDir string, integrationID string, creds *VaultCredentials) error {
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	cachePath := filepath.Join(cacheDir, vaultCacheFilePrefix+integrationID)

	if err := os.WriteFile(cachePath, []byte(encoded), 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

func loadVaultCredentials(cacheDir string, integrationID string) (*VaultCredentials, error) {
	cachePath := filepath.Join(cacheDir, vaultCacheFilePrefix+integrationID)

	encoded, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, fmt.Errorf("no cached credentials for vault: %w", err)
	}

	data, err := base64.StdEncoding.DecodeString(string(encoded))
	if err != nil {
		return nil, fmt.Errorf("failed to decode cached credentials: %w", err)
	}

	var creds VaultCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached credentials: %w", err)
	}

	return &creds, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./pkg/services/ -run TestVaultCredentials -v`
Expected: PASS

**Step 5: Commit**

```
feat(vault): add credential caching for vault commands
```

---

### Task 2: Vault Service — Session Discovery and Credential Resolution

Functions to find active vault sessions and resolve credentials (cache or API).

**Files:**
- Modify: `pkg/services/vault.go`
- Test: `pkg/services/vault_test.go`

**Step 1: Write tests for path parsing**

```go
// append to pkg/services/vault_test.go

func TestParseVaultPath(t *testing.T) {
	tests := []struct {
		input     string
		mount     string
		secretPath string
		apiPath   string
	}{
		{"kv/db/prod", "kv", "db/prod", "kv/data/db/prod"},
		{"kv/simple", "kv", "simple", "kv/data/simple"},
		{"secret/nested/deep/path", "secret", "nested/deep/path", "secret/data/nested/deep/path"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mount, secretPath, err := ParseVaultPath(tt.input)
			if err != nil {
				t.Fatalf("ParseVaultPath(%q) error: %v", tt.input, err)
			}
			if mount != tt.mount {
				t.Errorf("mount = %q, want %q", mount, tt.mount)
			}
			if secretPath != tt.secretPath {
				t.Errorf("secretPath = %q, want %q", secretPath, tt.secretPath)
			}
			apiPath := VaultKVDataPath(mount, secretPath)
			if apiPath != tt.apiPath {
				t.Errorf("apiPath = %q, want %q", apiPath, tt.apiPath)
			}
		})
	}
}

func TestParseVaultPath_Invalid(t *testing.T) {
	_, _, err := ParseVaultPath("noseparator")
	if err == nil {
		t.Fatal("expected error for path without separator")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./pkg/services/ -run TestParseVaultPath -v`
Expected: FAIL

**Step 3: Implement path parsing and session discovery**

Add to `pkg/services/vault.go`:

```go
import (
	"context"
	"strings"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
)

const (
	vaultSecretSessionType     = "session-apono-vault-secret"
	vaultManagementSessionType = "session-apono-vault-management"
)

// ParseVaultPath splits a user-facing path like "kv/db/prod" into mount ("kv") and secret path ("db/prod").
func ParseVaultPath(path string) (mount string, secretPath string, err error) {
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid vault path %q: expected format <mount>/<secret>, e.g. kv/my-secret", path)
	}
	return parts[0], parts[1], nil
}

// VaultKVDataPath constructs the KV v2 API data path.
func VaultKVDataPath(mount string, secretPath string) string {
	return mount + "/data/" + secretPath
}

// VaultKVMetadataPath constructs the KV v2 API metadata path for listing.
func VaultKVMetadataPath(mount string) string {
	return mount + "/metadata/"
}

// FindVaultSession finds an active session for the given integration ID and session type.
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

// ResolveVaultCredentials gets credentials from cache or from the access details API.
func ResolveVaultCredentials(ctx context.Context, client *aponoapi.AponoClient, integrationID string, session *clientapi.AccessSessionClientModel) (*VaultCredentials, error) {
	cacheDir := defaultCacheDir()

	// Try cache first
	cached, err := loadVaultCredentials(cacheDir, integrationID)
	if err == nil {
		return cached, nil
	}

	// Fall back to access details API
	accessDetails, _, err := client.ClientAPI.AccessSessionsAPI.GetAccessSessionAccessDetails(ctx, session.Id).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get access details for session %s: %w", session.Id, err)
	}

	jsonData := accessDetails.GetJson()
	vaultAddress, _ := jsonData["vault_address"].(string)
	username, _ := jsonData["username"].(string)
	password, _ := jsonData["password"].(string)

	if password == "" {
		return nil, fmt.Errorf("credentials not available. Run 'apono access reset-credentials %s' then retry", session.Id)
	}

	creds := &VaultCredentials{
		VaultAddress: vaultAddress,
		Username:     username,
		Password:     password,
	}

	// Save to cache (ignore error — caching is best-effort)
	_ = saveVaultCredentials(cacheDir, integrationID, creds)

	return creds, nil
}
```

**Step 4: Run tests**

Run: `go test ./pkg/services/ -run TestParseVaultPath -v`
Expected: PASS

**Step 5: Commit**

```
feat(vault): add session discovery and credential resolution
```

---

### Task 3: Vault Service — Vault HTTP Client

HTTP operations: login, read, write, list, delete secrets.

**Files:**
- Modify: `pkg/services/vault.go`
- Test: `pkg/services/vault_test.go`

**Step 1: Write test for VaultClient path construction**

```go
// append to pkg/services/vault_test.go

func TestVaultKVDataPath(t *testing.T) {
	result := VaultKVDataPath("kv", "db/prod")
	expected := "kv/data/db/prod"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestVaultKVMetadataPath(t *testing.T) {
	result := VaultKVMetadataPath("kv")
	expected := "kv/metadata/"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}
```

**Step 2: Run tests — should pass immediately (functions already exist)**

Run: `go test ./pkg/services/ -run TestVaultKV -v`
Expected: PASS

**Step 3: Implement VaultClient**

Add to `pkg/services/vault.go`:

```go
import (
	"bytes"
	"io"
	"net/http"
)

// VaultClient performs direct HTTP calls to the vault.
type VaultClient struct {
	Address string
	Token   string
}

// Login authenticates to vault via userpass and returns a VaultClient with a token.
func VaultLogin(address string, username string, password string) (*VaultClient, error) {
	url := fmt.Sprintf("%s/v1/auth/userpass/login/%s", strings.TrimRight(address, "/"), username)

	body, _ := json.Marshal(map[string]string{"password": password})
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to vault at %s: %w", address, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to authenticate to vault at %s: %s", address, string(respBody))
	}

	var result struct {
		Auth struct {
			ClientToken string `json:"client_token"`
		} `json:"auth"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse vault login response: %w", err)
	}

	return &VaultClient{Address: address, Token: result.Auth.ClientToken}, nil
}

// Read reads a secret at the given API path.
func (vc *VaultClient) Read(apiPath string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/v1/%s", strings.TrimRight(vc.Address, "/"), apiPath)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-Vault-Token", vc.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vault request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("secret not found at path %q", apiPath)
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("vault operation failed: %s", string(respBody))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse vault response: %w", err)
	}

	return result, nil
}

// Write writes data to the given API path (create/update).
func (vc *VaultClient) Write(apiPath string, data map[string]interface{}) error {
	url := fmt.Sprintf("%s/v1/%s", strings.TrimRight(vc.Address, "/"), apiPath)

	body, _ := json.Marshal(map[string]interface{}{"data": data})
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("X-Vault-Token", vc.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("vault request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("vault operation failed: %s", string(respBody))
	}

	return nil
}

// List lists secrets at the given metadata path.
func (vc *VaultClient) List(metadataPath string) ([]string, error) {
	url := fmt.Sprintf("%s/v1/%s", strings.TrimRight(vc.Address, "/"), metadataPath)

	req, _ := http.NewRequest("LIST", url, nil)
	req.Header.Set("X-Vault-Token", vc.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vault request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return []string{}, nil
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("vault operation failed: %s", string(respBody))
	}

	var result struct {
		Data struct {
			Keys []string `json:"keys"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse vault response: %w", err)
	}

	return result.Data.Keys, nil
}

// Delete deletes a secret at the given API path.
func (vc *VaultClient) Delete(apiPath string) error {
	url := fmt.Sprintf("%s/v1/%s", strings.TrimRight(vc.Address, "/"), apiPath)

	req, _ := http.NewRequest("DELETE", url, nil)
	req.Header.Set("X-Vault-Token", vc.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("vault request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("vault operation failed: %s", string(respBody))
	}

	return nil
}
```

**Step 4: Run all vault service tests**

Run: `go test ./pkg/services/ -run TestVault -v`
Expected: PASS

**Step 5: Commit**

```
feat(vault): add vault HTTP client for KV v2 operations
```

---

### Task 4: Command Scaffold — `apono vault` parent + `fetch`

First command: the parent `vault` group and `fetch` subcommand (DVL-8149).

**Files:**
- Create: `pkg/commands/vault/configurator.go`
- Create: `pkg/commands/vault/actions/vault.go`
- Create: `pkg/commands/vault/actions/fetch.go`
- Modify: `pkg/commands/apono/runner.go:29-35` — add vault configurator

**Step 1: Create parent command**

```go
// pkg/commands/vault/actions/vault.go
package actions

import (
	"github.com/apono-io/apono-cli/pkg/groups"
	"github.com/spf13/cobra"
)

func Vault() *cobra.Command {
	return &cobra.Command{
		Use:     "vault",
		Short:   "Manage Apono vault secrets",
		GroupID: groups.ManagementCommandsGroup.ID,
	}
}
```

**Step 2: Create fetch command**

```go
// pkg/commands/vault/actions/fetch.go
package actions

import (
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/utils"
	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
)

func VaultFetch() *cobra.Command {
	format := new(utils.Format)
	var vaultID string

	cmd := &cobra.Command{
		Use:   "fetch <path>",
		Short: "Fetch a secret from the vault",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			secretPath := args[0]

			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			integration, err := services.GetIntegrationByIDOrByTypeAndName(cmd.Context(), client, vaultID)
			if err != nil {
				return fmt.Errorf("vault %q not found: %w", vaultID, err)
			}

			session, err := services.FindVaultSession(cmd.Context(), client, integration.Id, services.VaultSecretSessionType)
			if err != nil {
				return err
			}
			if session == nil {
				return fmt.Errorf("no active vault access for %q. Request access via 'apono access' or the Apono portal", vaultID)
			}

			creds, err := services.ResolveVaultCredentials(cmd.Context(), client, integration.Id, session)
			if err != nil {
				return err
			}

			vc, err := services.VaultLogin(creds.VaultAddress, creds.Username, creds.Password)
			if err != nil {
				return err
			}

			mount, secretName, err := services.ParseVaultPath(secretPath)
			if err != nil {
				return err
			}

			apiPath := services.VaultKVDataPath(mount, secretName)
			result, err := vc.Read(apiPath)
			if err != nil {
				return fmt.Errorf("secret %q not found in vault %q: %w", secretPath, vaultID, err)
			}

			return printFetchResult(cmd, result, format)
		},
	}

	flags := cmd.Flags()
	utils.AddFormatFlag(flags, format)
	flags.StringVar(&vaultID, "vault-id", "", "Vault integration ID or name (required)")
	_ = cmd.MarkFlagRequired("vault-id")

	return cmd
}

func printFetchResult(cmd *cobra.Command, result map[string]interface{}, format *utils.Format) error {
	// Extract the "data" field from KV v2 response
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		data = result
	}
	secretData, ok := data["data"].(map[string]interface{})
	if ok {
		data = secretData
	}

	switch *format {
	case utils.JSONFormat:
		return utils.PrintObjectsAsJSON(cmd.OutOrStdout(), data)
	default:
		table := uitable.New()
		table.AddRow("KEY", "VALUE")
		for k, v := range data {
			table.AddRow(k, fmt.Sprintf("%v", v))
		}
		_, err := fmt.Fprintln(cmd.OutOrStdout(), table)
		return err
	}
}
```

**Step 3: Create configurator**

```go
// pkg/commands/vault/configurator.go
package vault

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/commands/vault/actions"
)

type Configurator struct{}

func (c *Configurator) ConfigureCommands(rootCmd *cobra.Command) error {
	vaultCmd := actions.Vault()
	rootCmd.AddCommand(vaultCmd)

	vaultCmd.AddCommand(actions.VaultFetch())

	return nil
}
```

**Step 4: Register in runner**

Modify `pkg/commands/apono/runner.go`:
- Add import: `"github.com/apono-io/apono-cli/pkg/commands/vault"`
- Add to configurators slice (after `&mcp.Configurator{}`): `&vault.Configurator{}`

**Step 5: Build and verify**

Run: `go build ./cmd/apono/`
Run: `go run ./cmd/apono/ vault --help`
Run: `go run ./cmd/apono/ vault fetch --help`
Expected: Help output showing vault commands.

**Step 6: Commit**

```
feat(vault): add vault parent command and fetch subcommand (DVL-8149)
```

---

### Task 5: Management Commands — `list`, `create`, `update`, `delete`

DVL-8148 commands.

**Files:**
- Create: `pkg/commands/vault/actions/list.go`
- Create: `pkg/commands/vault/actions/create.go`
- Create: `pkg/commands/vault/actions/update.go`
- Create: `pkg/commands/vault/actions/delete.go`
- Modify: `pkg/commands/vault/configurator.go` — register new subcommands

**Step 1: Create list command**

```go
// pkg/commands/vault/actions/list.go
package actions

import (
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/utils"
	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
)

func VaultList() *cobra.Command {
	format := new(utils.Format)
	var vaultID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List secrets in the vault",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			integration, err := services.GetIntegrationByIDOrByTypeAndName(cmd.Context(), client, vaultID)
			if err != nil {
				return fmt.Errorf("vault %q not found: %w", vaultID, err)
			}

			session, err := services.FindVaultSession(cmd.Context(), client, integration.Id, services.VaultManagementSessionType)
			if err != nil {
				return err
			}
			if session == nil {
				return fmt.Errorf("no active vault management access for %q. Request management access via 'apono access' or the Apono portal", vaultID)
			}

			creds, err := services.ResolveVaultCredentials(cmd.Context(), client, integration.Id, session)
			if err != nil {
				return err
			}

			vc, err := services.VaultLogin(creds.VaultAddress, creds.Username, creds.Password)
			if err != nil {
				return err
			}

			// Get mount name from session details JSON
			accessDetails, _, err := client.ClientAPI.AccessSessionsAPI.GetAccessSessionAccessDetails(cmd.Context(), session.Id).Execute()
			if err != nil {
				return fmt.Errorf("failed to get session details: %w", err)
			}
			mountName, _ := accessDetails.GetJson()["mount_name"].(string)
			if mountName == "" {
				mountName = "kv"
			}

			metadataPath := services.VaultKVMetadataPath(mountName)
			keys, err := vc.List(metadataPath)
			if err != nil {
				return err
			}

			if len(keys) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No secrets found")
				return nil
			}

			return printListResult(cmd, keys, mountName, format)
		},
	}

	flags := cmd.Flags()
	utils.AddFormatFlag(flags, format)
	flags.StringVar(&vaultID, "vault-id", "", "Vault integration ID or name (required)")
	_ = cmd.MarkFlagRequired("vault-id")

	return cmd
}

func printListResult(cmd *cobra.Command, keys []string, mount string, format *utils.Format) error {
	switch *format {
	case utils.JSONFormat:
		return utils.PrintObjectsAsJSON(cmd.OutOrStdout(), keys)
	default:
		table := uitable.New()
		table.AddRow("SECRET PATH")
		for _, key := range keys {
			table.AddRow(mount + "/" + key)
		}
		_, err := fmt.Fprintln(cmd.OutOrStdout(), table)
		return err
	}
}
```

**Step 2: Create create command**

```go
// pkg/commands/vault/actions/create.go
package actions

import (
	"encoding/json"
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/spf13/cobra"
)

func VaultCreate() *cobra.Command {
	var vaultID string
	var value string

	cmd := &cobra.Command{
		Use:   "create <path>",
		Short: "Create a new secret in the vault",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			secretPath := args[0]

			var secretData map[string]interface{}
			if err := json.Unmarshal([]byte(value), &secretData); err != nil {
				return fmt.Errorf("invalid JSON value: %w", err)
			}

			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			integration, err := services.GetIntegrationByIDOrByTypeAndName(cmd.Context(), client, vaultID)
			if err != nil {
				return fmt.Errorf("vault %q not found: %w", vaultID, err)
			}

			session, err := services.FindVaultSession(cmd.Context(), client, integration.Id, services.VaultManagementSessionType)
			if err != nil {
				return err
			}
			if session == nil {
				return fmt.Errorf("no active vault management access for %q. Request management access via 'apono access' or the Apono portal", vaultID)
			}

			creds, err := services.ResolveVaultCredentials(cmd.Context(), client, integration.Id, session)
			if err != nil {
				return err
			}

			vc, err := services.VaultLogin(creds.VaultAddress, creds.Username, creds.Password)
			if err != nil {
				return err
			}

			mount, secretName, err := services.ParseVaultPath(secretPath)
			if err != nil {
				return err
			}

			apiPath := services.VaultKVDataPath(mount, secretName)
			if err := vc.Write(apiPath, secretData); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Secret %q created successfully\n", secretPath)
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&vaultID, "vault-id", "", "Vault integration ID or name (required)")
	flags.StringVar(&value, "value", "", "Secret value as JSON string (required)")
	_ = cmd.MarkFlagRequired("vault-id")
	_ = cmd.MarkFlagRequired("value")

	return cmd
}
```

**Step 3: Create update command**

```go
// pkg/commands/vault/actions/update.go
package actions

import (
	"encoding/json"
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/spf13/cobra"
)

func VaultUpdate() *cobra.Command {
	var vaultID string
	var value string

	cmd := &cobra.Command{
		Use:   "update <path>",
		Short: "Update an existing secret in the vault",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			secretPath := args[0]

			var secretData map[string]interface{}
			if err := json.Unmarshal([]byte(value), &secretData); err != nil {
				return fmt.Errorf("invalid JSON value: %w", err)
			}

			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			integration, err := services.GetIntegrationByIDOrByTypeAndName(cmd.Context(), client, vaultID)
			if err != nil {
				return fmt.Errorf("vault %q not found: %w", vaultID, err)
			}

			session, err := services.FindVaultSession(cmd.Context(), client, integration.Id, services.VaultManagementSessionType)
			if err != nil {
				return err
			}
			if session == nil {
				return fmt.Errorf("no active vault management access for %q. Request management access via 'apono access' or the Apono portal", vaultID)
			}

			creds, err := services.ResolveVaultCredentials(cmd.Context(), client, integration.Id, session)
			if err != nil {
				return err
			}

			vc, err := services.VaultLogin(creds.VaultAddress, creds.Username, creds.Password)
			if err != nil {
				return err
			}

			mount, secretName, err := services.ParseVaultPath(secretPath)
			if err != nil {
				return err
			}

			apiPath := services.VaultKVDataPath(mount, secretName)
			if err := vc.Write(apiPath, secretData); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Secret %q updated successfully\n", secretPath)
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&vaultID, "vault-id", "", "Vault integration ID or name (required)")
	flags.StringVar(&value, "value", "", "Secret value as JSON string (required)")
	_ = cmd.MarkFlagRequired("vault-id")
	_ = cmd.MarkFlagRequired("value")

	return cmd
}
```

**Step 4: Create delete command**

```go
// pkg/commands/vault/actions/delete.go
package actions

import (
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/spf13/cobra"
)

func VaultDelete() *cobra.Command {
	var vaultID string

	cmd := &cobra.Command{
		Use:   "delete <path>",
		Short: "Delete a secret from the vault",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			secretPath := args[0]

			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			integration, err := services.GetIntegrationByIDOrByTypeAndName(cmd.Context(), client, vaultID)
			if err != nil {
				return fmt.Errorf("vault %q not found: %w", vaultID, err)
			}

			session, err := services.FindVaultSession(cmd.Context(), client, integration.Id, services.VaultManagementSessionType)
			if err != nil {
				return err
			}
			if session == nil {
				return fmt.Errorf("no active vault management access for %q. Request management access via 'apono access' or the Apono portal", vaultID)
			}

			creds, err := services.ResolveVaultCredentials(cmd.Context(), client, integration.Id, session)
			if err != nil {
				return err
			}

			vc, err := services.VaultLogin(creds.VaultAddress, creds.Username, creds.Password)
			if err != nil {
				return err
			}

			mount, secretName, err := services.ParseVaultPath(secretPath)
			if err != nil {
				return err
			}

			apiPath := services.VaultKVDataPath(mount, secretName)
			if err := vc.Delete(apiPath); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Secret %q deleted successfully\n", secretPath)
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&vaultID, "vault-id", "", "Vault integration ID or name (required)")
	_ = cmd.MarkFlagRequired("vault-id")

	return cmd
}
```

**Step 5: Update configurator**

Modify `pkg/commands/vault/configurator.go` to register all commands:

```go
vaultCmd.AddCommand(actions.VaultFetch())
vaultCmd.AddCommand(actions.VaultList())
vaultCmd.AddCommand(actions.VaultCreate())
vaultCmd.AddCommand(actions.VaultUpdate())
vaultCmd.AddCommand(actions.VaultDelete())
```

**Step 6: Build and verify**

Run: `go build ./cmd/apono/`
Run: `go run ./cmd/apono/ vault --help`
Expected: Shows all 5 subcommands.

**Step 7: Commit**

```
feat(vault): add list, create, update, delete commands (DVL-8148)
```

---

### Task 6: Export Session Type Constants and Lint/Test

Make sure the exported constants are correct and everything compiles and passes lint.

**Files:**
- Modify: `pkg/services/vault.go` — ensure `VaultSecretSessionType` and `VaultManagementSessionType` are exported (the const names in Task 2 used unexported names, fix that)

**Step 1: Verify constant names are exported**

In `pkg/services/vault.go`, ensure:
```go
const (
	VaultSecretSessionType     = "session-apono-vault-secret"
	VaultManagementSessionType = "session-apono-vault-management"
)
```

**Step 2: Run full test suite**

Run: `go test ./...`
Expected: All tests pass.

**Step 3: Run linter**

Run: `make lint`
Expected: No errors (or only pre-existing ones).

**Step 4: Build**

Run: `make build`
Expected: Successful build.

**Step 5: Commit**

```
chore(vault): clean up exports, lint, and build verification
```

---

### Task 7: Integration Config Update

Update the vault integration config in the integrations repo to support credential flow.

**Files:**
- Modify: `~/apono/integrations/configs/templates/apono-vault/apono-vault.json`

**Step 1: Update session types**

For both `session-apono-vault-secret` and `session-apono-vault-management`, update:
- `cred_params`: add `"password"`
- `common_params`: add `"session_id"`
- Add `credentials` template in `access_details_templates`

The `credentials` template for secret session:
```json
{
  "vault_address": "{{{params.vault_address}}}",
  "username": "{{{cred_params.username}}}",
  "password": "{{{cred_params.password}}}",
  "secret_name": "{{{params.secret_name}}}"
}
```

The `credentials` template for management session:
```json
{
  "vault_address": "{{{params.vault_address}}}",
  "username": "{{{cred_params.username}}}",
  "password": "{{{cred_params.password}}}",
  "mount_name": "{{{params.mount_name}}}"
}
```

**Step 2: Verify JSON is valid**

Run: `python3 -m json.tool ~/apono/integrations/configs/templates/apono-vault/apono-vault.json`
Expected: Valid JSON output.

**Step 3: Commit (in integrations repo)**

```
feat(apono-vault): add password cred_param and credentials template for CLI support
```

---
