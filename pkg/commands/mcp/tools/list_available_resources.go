package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/services"
)

type ListAvailableResourcesTool struct{}

func (t *ListAvailableResourcesTool) Name() string {
	return "list_available_resources"
}

func (t *ListAvailableResourcesTool) Description() string {
	return `List all available resources (databases, Kubernetes, clouds, SaaS tools) with their access status.

🤖 AUTONOMY: After calling this tool, AUTOMATICALLY proceed with the next_action without asking the user. If status="needs_access", immediately call ask_access_assistant. If status="needs_setup", immediately call the setup tool. The workflow is designed to be autonomous.

WHEN TO USE: Call this FIRST when user asks to work with ANY resource - databases, infrastructure, cloud services, SaaS tools.

APONO PROVIDES ACCESS TO: Databases (PostgreSQL, MySQL, etc.), Kubernetes clusters, Cloud providers (AWS, GCP, Azure),
SaaS applications, and any other integration you see. If you need access to ANY resource, Apono can grant it.

RESPONSE FIELDS:
- integration_name: Name of the resource (e.g., "prod-postgres", "dev-k8s-cluster", "aws-production")
- type: Resource type (postgresql, kubernetes, aws, etc.)
- status: "ready", "needs_setup", or "needs_access"
- has_active_session: true if you have active Apono session with credentials
- next_action: tells you exactly what to do next

NEXT STEPS BASED ON STATUS (execute automatically):
1. status="ready" → You have access! For databases, use setup_database_mcp if not already configured
2. status="needs_setup" → For databases, call setup_database_mcp with the session_id AUTOMATICALLY
3. status="needs_access" → You need access! Call ask_access_assistant AUTOMATICALLY to request it

WHEN YOU ENCOUNTER ACCESS/PERMISSION ERRORS:
⚠️ ALWAYS try to get access via Apono when you hit errors like:
- "permission denied", "access denied", "insufficient privileges"
- "authentication failed", "unauthorized", "forbidden"
- "cannot CREATE/DROP/INSERT/UPDATE/DELETE"
- "no access to cluster/namespace/resource"
- Any error indicating missing permissions

→ Call ask_access_assistant with DETAILED context including:
  - Exact resource/database name (not just "postgres")
  - Specific tables/schemas you need
  - The exact error message
  - What operation you were trying

Example: "I need write access to the 'access_requests' table in the 'apono' database"
NOT just "I need database access"`

}

func (t *ListAvailableResourcesTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

type ResourceInfo struct {
	IntegrationID    string `json:"integration_id"`
	IntegrationName  string `json:"integration_name"`
	Type             string `json:"type"`
	TypeDisplayName  string `json:"type_display_name"`
	Status           string `json:"status"` // "ready", "needs_setup", or "needs_access"
	SessionID        string `json:"session_id,omitempty"`
	SessionName      string `json:"session_name,omitempty"`
	HasActiveSession bool   `json:"has_active_session"` // Has Apono session
	McpConfigured    bool   `json:"mcp_configured"`     // MCP is in Cursor config
	CanConnect       bool   `json:"can_connect"`        // Both are true - can query immediately
	NextAction       string `json:"next_action"`        // "query", "setup_mcp", or "request_access"
}

type CursorMCPConfig struct {
	MCPServers map[string]interface{} `json:"mcpServers"`
}

func (t *ListAvailableResourcesTool) Execute(ctx context.Context, client *aponoapi.AponoClient, arguments json.RawMessage) (interface{}, error) {
	// Get all integrations
	integrations, err := services.ListIntegrations(ctx, client)
	if err != nil {
		return nil, err
	}

	// Get all active sessions
	sessions, err := services.ListAccessSessions(ctx, client, []string{}, []string{}, []string{})
	if err != nil {
		return nil, err
	}

	// Load Cursor MCP config to check what's configured
	configuredMCPs := loadCursorMCPConfig()

	// Create a map of integration ID -> active session
	sessionMap := make(map[string]struct {
		id   string
		name string
	})
	for _, session := range sessions {
		sessionMap[session.Integration.Id] = struct {
			id   string
			name string
		}{
			id:   session.Id,
			name: session.Name,
		}
	}

	// Build resource list
	resources := make([]ResourceInfo, 0, len(integrations))
	readyCount := 0
	needsSetupCount := 0
	needsAccessCount := 0

	for _, integration := range integrations {
		resource := ResourceInfo{
			IntegrationID:   integration.Id,
			IntegrationName: integration.Name,
			Type:            integration.Type,
			TypeDisplayName: integration.TypeDisplayName,
		}

		// Check if there's an active session
		if sessionInfo, hasSession := sessionMap[integration.Id]; hasSession {
			resource.HasActiveSession = true
			resource.SessionID = sessionInfo.id
			resource.SessionName = sessionInfo.name

			// Check if MCP is configured - use integration name for MCP server name
			mcpServerName := fmt.Sprintf("postgres-%s", sanitizeName(integration.Name))
			_, mcpExists := configuredMCPs[mcpServerName]
			resource.McpConfigured = mcpExists

			if mcpExists {
				// Both session and MCP exist - ready to query!
				resource.Status = "ready"
				resource.CanConnect = true
				resource.NextAction = "query"
				readyCount++
			} else {
				// Has session but MCP not configured - need to setup
				resource.Status = "needs_setup"
				resource.CanConnect = false
				resource.NextAction = "setup_mcp"
				needsSetupCount++
			}
		} else {
			// No active session - need to request access
			resource.HasActiveSession = false
			resource.McpConfigured = false
			resource.Status = "needs_access"
			resource.CanConnect = false
			resource.NextAction = "request_access"
			needsAccessCount++
		}

		resources = append(resources, resource)
	}

	return map[string]interface{}{
		"resources":          resources,
		"total":              len(resources),
		"ready_count":        readyCount,
		"needs_setup_count":  needsSetupCount,
		"needs_access_count": needsAccessCount,
	}, nil
}

func loadCursorMCPConfig() map[string]bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return make(map[string]bool)
	}

	configPath := filepath.Join(homeDir, ".cursor", "mcp.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return make(map[string]bool)
	}

	var config CursorMCPConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return make(map[string]bool)
	}

	// Convert to a simple map for lookup
	result := make(map[string]bool)
	for name := range config.MCPServers {
		result[name] = true
	}
	return result
}
