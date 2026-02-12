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
	"gopkg.in/yaml.v3"
)

type SetupDatabaseMCPV2Tool struct{}

func (t *SetupDatabaseMCPV2Tool) Name() string {
	return "setup_database_mcp_v2"
}

func (t *SetupDatabaseMCPV2Tool) Description() string {
	return `Configure PostgreSQL target for apono-mcp-mux.

⭐ WHEN TO USE:
1. When status="needs_setup" in list_available_resources (has session but MCP not configured)
2. When PostgreSQL queries fail with authentication errors (stale credentials - refresh needed)
3. When using apono-mcp-mux proxy for centralized MCP management

PREREQUISITES:
- Resource must have has_active_session=true
- You need the session_id from list_available_resources response
- apono-mcp-mux must be configured in Cursor (manual one-time setup)

WHAT IT DOES:
1. Fetches fresh database credentials from the active Apono session
2. Writes target configuration to ~/.apono/mcp-mux/targets.yaml
3. Returns target_id for use with _proxy__init_target

TWO-STEP WORKFLOW:
1. Call this tool (setup_database_mcp_v2) to configure the target
2. Call _proxy__init_target with the returned target_id to activate the MCP server
   Example: _proxy__init_target(target_id="local-postgres")

RESPONSE:
- success: true/false
- target_id: Target ID to use with _proxy__init_target (e.g., "local-postgres")
- message: Status message
- next_step: Instructions on what to do next

AFTER CALLING THIS TOOL:
⚠️ CRITICAL - You must call _proxy__init_target before using the database!
1. Call _proxy__init_target with target_id from response
2. Wait for initialization to complete
3. PostgreSQL MCP tools will become available
4. You can then query the database

NOTE: If you get "password authentication failed" errors, call this tool again to refresh credentials.`
}

func (t *SetupDatabaseMCPV2Tool) InputSchema() map[string]interface{} {
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

type SetupDatabaseMCPV2Input struct {
	SessionID string `json:"session_id"`
}

// MuxTarget represents a single target in the mux configuration
type MuxTarget struct {
	ID          string            `yaml:"id"`
	Name        string            `yaml:"name"`
	Type        string            `yaml:"type"`
	AutoInit    bool              `yaml:"auto_init,omitempty"`
	Credentials map[string]string `yaml:"credentials"`
	Env         map[string]string `yaml:"env,omitempty"`
}

// AponoApprovalConfig represents Apono approval settings
type AponoApprovalConfig struct {
	Profile     string `yaml:"profile"`
	AgentUserID string `yaml:"agent_user_id"`
}

// MuxUser represents a user with their targets
type MuxUser struct {
	UserID        string               `yaml:"user_id"`
	AponoApproval *AponoApprovalConfig `yaml:"apono_approval,omitempty"`
	Targets       []MuxTarget          `yaml:"targets"`
}

// MuxTargetsConfig represents the complete targets.yaml structure
type MuxTargetsConfig struct {
	Users []MuxUser `yaml:"users"`
}

func (t *SetupDatabaseMCPV2Tool) Execute(ctx context.Context, client *aponoapi.AponoClient, arguments json.RawMessage) (interface{}, error) {
	var input SetupDatabaseMCPV2Input
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
			if err := resetCredentialsV2(ctx, client, input.SessionID); err != nil {
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
	if !isPostgreSQLSessionV2(creds) {
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

	// Get targets file path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Check for custom targets file path from env var
	targetsPath := os.Getenv("APONO_MCP_MUX_TARGETS_FILE")
	if targetsPath == "" {
		targetsPath = filepath.Join(homeDir, ".apono", "mcp-mux", "targets.yaml")
	}

	// Get user ID from env var, default to "cursor-user"
	userID := os.Getenv("APONO_MCP_USER_ID")
	if userID == "" {
		userID = "cursor-user"
	}

	// Load existing targets or create new
	var config MuxTargetsConfig
	if data, err := os.ReadFile(targetsPath); err == nil {
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse targets.yaml: %w", err)
		}
	} else {
		config = MuxTargetsConfig{
			Users: []MuxUser{},
		}
	}

	// Find or create user
	var userIdx int = -1
	for i, user := range config.Users {
		if user.UserID == userID {
			userIdx = i
			break
		}
	}

	if userIdx == -1 {
		// Create new user
		config.Users = append(config.Users, MuxUser{
			UserID:  userID,
			Targets: []MuxTarget{},
		})
		userIdx = len(config.Users) - 1
	}

	// Target ID is the sanitized integration name
	targetID := sanitizeNameV2(integrationName)

	// Create or update target
	target := MuxTarget{
		ID:   targetID,
		Name: fmt.Sprintf("Apono: %s", integrationName),
		Type: "postgres",
		Credentials: map[string]string{
			"database_url": connectionString,
		},
	}

	// Find existing target or add new
	targetIdx := -1
	for i, t := range config.Users[userIdx].Targets {
		if t.ID == targetID {
			targetIdx = i
			break
		}
	}

	if targetIdx == -1 {
		// Add new target
		config.Users[userIdx].Targets = append(config.Users[userIdx].Targets, target)
	} else {
		// Update existing target
		config.Users[userIdx].Targets[targetIdx] = target
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(targetsPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Write updated config
	configData, err := yaml.Marshal(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(targetsPath, configData, 0600); err != nil {
		return nil, fmt.Errorf("failed to write targets.yaml: %w", err)
	}

	fmt.Printf("[DEBUG] Target '%s' configured successfully in %s\n", targetID, targetsPath)

	return map[string]interface{}{
		"success":   true,
		"target_id": targetID,
		"database":  dbName,
		"host":      host,
		"message":   fmt.Sprintf("✓ Target '%s' configured in apono-mcp-mux!", targetID),
		"next_step": fmt.Sprintf("Call _proxy__init_target with target_id='%s' to activate the PostgreSQL MCP server", targetID),
	}, nil
}

func isPostgreSQLSessionV2(creds map[string]interface{}) bool {
	_, hasHost := creds["host"]
	_, hasPort := creds["port"]
	_, hasDbName := creds["db_name"]
	_, hasUsername := creds["username"]
	_, hasPassword := creds["password"]

	return hasHost && hasPort && hasDbName && hasUsername && hasPassword
}

// sanitizeNameV2 converts a name to a safe format for use in target IDs
// Example: "Local Postgres" -> "local-postgres"
func sanitizeNameV2(name string) string {
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

// resetCredentialsV2 resets the credentials for a session and waits for new credentials
func resetCredentialsV2(ctx context.Context, client *aponoapi.AponoClient, sessionID string) error {
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
