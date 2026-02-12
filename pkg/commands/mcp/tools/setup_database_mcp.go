package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/services"
)

type SetupDatabaseMCPTool struct{}

func (t *SetupDatabaseMCPTool) Name() string {
	return "setup_database_mcp"
}

func (t *SetupDatabaseMCPTool) Description() string {
	return `Configure or refresh PostgreSQL MCP connection in Cursor.

⚠️ NOTE: For apono-mcp-mux integration, use setup_database_mcp_v2 instead.
This tool directly modifies Cursor's mcp.json. Use setup_database_mcp_v2 for centralized MCP management via apono-mcp-mux proxy.

⭐ WHEN TO USE:
1. When status="needs_setup" in list_available_resources (has session but MCP not configured)
2. When PostgreSQL queries fail with authentication errors (stale credentials - refresh needed)

PREREQUISITES:
- Resource must have has_active_session=true
- You need the session_id from list_available_resources response

WHAT IT DOES:
1. Fetches fresh database credentials from the active Apono session
2. Updates/creates PostgreSQL MCP server entry in ~/.cursor/mcp.json
3. Cursor auto-reloads the config (no restart needed)
4. PostgreSQL MCP tools become available with valid credentials

RESPONSE:
- success: true/false
- message: Status message
- mcp_server_name: Name of the MCP server (e.g., "postgres-local-postgres")

⚠️ CRITICAL - AFTER SETUP, WAIT FOR CURSOR TO RELOAD:
1. After this tool succeeds, Cursor needs 2-5 seconds to reload MCP servers
2. WAIT 3 seconds before trying to use PostgreSQL MCP tools
3. If PostgreSQL tools (query_postgres, list_tables, etc.) are NOT available yet:
   - Tell user: "MCP server configured, waiting for Cursor to reload tools..."
   - Wait 3 more seconds
   - Try again
4. Once tools appear, you can query the database

AFTER TOOLS ARE AVAILABLE:
Use PostgreSQL MCP tools (list_tables, query, etc.) to work with the database.
The resource will show status="ready" in list_available_resources.

NOTE: If you get "password authentication failed" errors, call this tool again to refresh credentials.`
}

func (t *SetupDatabaseMCPTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"session_id": map[string]interface{}{
				"type":        "string",
				"description": "The ID of the active access session to setup MCP for",
			},
		},
		"required": []string{"session_id"},
	}
}

type SetupDatabaseMCPInput struct {
	SessionID string `json:"session_id"`
}

type MCPServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

type CursorConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

func (t *SetupDatabaseMCPTool) Execute(ctx context.Context, client *aponoapi.AponoClient, arguments json.RawMessage) (interface{}, error) {
	var input SetupDatabaseMCPInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if input.SessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	// Get the session to retrieve integration name
	sessions, err := services.ListAccessSessions(ctx, client, []string{}, []string{}, []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	var session *clientapi.AccessSessionClientModel
	for _, s := range sessions {
		if s.Id == input.SessionID {
			session = &s
			break
		}
	}

	if session == nil {
		return nil, fmt.Errorf("session not found: %s", input.SessionID)
	}

	integrationName := session.Integration.Name

	// Check if credentials need to be reset (expired/placeholder credentials)
	if session.Credentials.IsSet() {
		creds := session.Credentials.Get()
		fmt.Printf("[DEBUG] Current credential status: %s, CanReset: %v\n", creds.Status, creds.CanReset)

		// If credentials are not "new" and can be reset, reset them first
		if creds.Status != "new" && creds.CanReset {
			fmt.Println("[DEBUG] Credentials are stale, resetting...")
			if err := resetCredentials(ctx, client, input.SessionID); err != nil {
				return nil, fmt.Errorf("failed to reset credentials: %w", err)
			}
			fmt.Println("[DEBUG] Credentials reset successfully")
		}
	}

	// Get session credentials
	accessDetails, _, err := services.GetSessionDetails(ctx, client, input.SessionID, services.JSONOutputFormat)
	if err != nil {
		return nil, fmt.Errorf("failed to get session credentials: %w", err)
	}

	// Parse credentials
	var creds map[string]interface{}
	if err := json.Unmarshal([]byte(accessDetails), &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	// Check if it's a PostgreSQL session
	if !isPostgreSQLSession(creds) {
		return nil, fmt.Errorf("session is not a PostgreSQL database")
	}

	// Build connection string
	host := fmt.Sprintf("%v", creds["host"])
	port := fmt.Sprintf("%v", creds["port"])
	dbName := fmt.Sprintf("%v", creds["db_name"])
	username := fmt.Sprintf("%v", creds["username"])
	password := fmt.Sprintf("%v", creds["password"])

	// URL-encode username and password
	encodedUsername := url.QueryEscape(username)
	encodedPassword := url.QueryEscape(password)

	connectionString := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", encodedUsername, encodedPassword, host, port, dbName)

	// Get Cursor config path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".cursor")
	configPath := filepath.Join(configDir, "mcp.json")

	// Read existing config or create new
	var config CursorConfig
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse Cursor config: %w", err)
		}
	} else {
		config = CursorConfig{
			MCPServers: make(map[string]MCPServerConfig),
		}
	}

	// Add/update MCP server - use integration name for shorter, readable server name
	serverName := fmt.Sprintf("postgres-%s", sanitizeName(integrationName))

	// Preserve existing env vars if the server already exists
	existingEnv := make(map[string]string)
	if existingServer, exists := config.MCPServers[serverName]; exists && existingServer.Env != nil {
		existingEnv = existingServer.Env
	}

	config.MCPServers[serverName] = MCPServerConfig{
		Command: "npx",
		Args:    []string{"-y", "@henkey/postgres-mcp-server", "--connection-string", connectionString},
		Env:     existingEnv, // Preserve existing env vars
	}

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write updated config
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, configData, 0600); err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}

	// Wait for Cursor to reload the MCP configuration
	fmt.Println("[DEBUG] Waiting 3 seconds for Cursor to reload MCP configuration...")
	time.Sleep(3 * time.Second)

	return map[string]interface{}{
		"success":         true,
		"mcp_server_name": serverName,
		"database":        dbName,
		"host":            host,
		"message":         fmt.Sprintf("✓ PostgreSQL MCP server '%s' configured! Cursor has reloaded the configuration. PostgreSQL tools (query_postgres, list_tables, etc.) should now be available.", serverName),
		"next_steps":      "You can now use PostgreSQL MCP tools to query the database. If tools are not available yet, wait a moment and try again.",
	}, nil
}

func isPostgreSQLSession(creds map[string]interface{}) bool {
	_, hasHost := creds["host"]
	_, hasPort := creds["port"]
	_, hasDbName := creds["db_name"]
	_, hasUsername := creds["username"]
	_, hasPassword := creds["password"]

	return hasHost && hasPort && hasDbName && hasUsername && hasPassword
}

// sanitizeName converts a name to a safe format for use in config keys
// Example: "Local Postgres" -> "local-postgres"
func sanitizeName(name string) string {
	// Convert to lowercase
	result := ""
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result += string(r)
		} else if r >= 'A' && r <= 'Z' {
			result += string(r + 32) // Convert to lowercase
		} else if r == ' ' || r == '_' {
			result += "-"
		}
		// Skip other special characters
	}
	return result
}

// resetCredentials resets the credentials for a session and waits for new credentials
func resetCredentials(ctx context.Context, client *aponoapi.AponoClient, sessionID string) error {
	const (
		newCredentialsStatus = "new"
		maxWaitTime          = 30 * time.Second
	)

	// Request credential reset
	_, _, err := client.ClientAPI.AccessSessionsAPI.ResetAccessSessionCredentials(ctx, sessionID).Execute()
	if err != nil {
		return fmt.Errorf("failed to reset credentials: %w", err)
	}

	fmt.Println("[DEBUG] Waiting for new credentials...")

	// Wait for credentials to be reset
	startTime := time.Now()
	for {
		session, _, err := client.ClientAPI.AccessSessionsAPI.GetAccessSession(ctx, sessionID).Execute()
		if err != nil {
			return fmt.Errorf("failed to get session status: %w", err)
		}

		if session.Credentials.IsSet() && session.Credentials.Get().Status == newCredentialsStatus {
			return nil // Credentials are fresh now
		}

		time.Sleep(1 * time.Second)

		if time.Now().After(startTime.Add(maxWaitTime)) {
			return fmt.Errorf("timeout while waiting for credentials to reset")
		}
	}
}
