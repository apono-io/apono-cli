package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/apono-io/apono-cli/pkg/commands/mcp/targets"
	"github.com/apono-io/apono-cli/pkg/utils"
)

const (
	// BuiltinBackendID is the prefix for proxy-level built-in tools
	BuiltinBackendID = "_proxy"
)

// BuiltinToolsHandler provides proxy-level built-in tools
type BuiltinToolsHandler struct {
	manager *LocalProxyManager
}

// NewBuiltinToolsHandler creates a new built-in tools handler
func NewBuiltinToolsHandler(manager *LocalProxyManager) *BuiltinToolsHandler {
	return &BuiltinToolsHandler{
		manager: manager,
	}
}

// GetTools returns the built-in tool definitions
func (h *BuiltinToolsHandler) GetTools() []Tool {
	return []Tool{
		{
			Name:        "list_targets",
			Description: "List all available database targets with their access status. Returns targets from both Apono sessions and local targets.yaml file. Each target has a status: ready (can init immediately), needs_access (will auto-request access), or pending (access request in progress).",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
			BackendID: BuiltinBackendID,
		},
		{
			Name:        "init_target",
			Description: "Initialize a target MCP server. Spawns a subprocess MCP server with credentials from your active Apono session or targets.yaml. If the target needs access, it will automatically request access and wait for approval. After initialization, the target's tools will appear in tools/list with the target ID as prefix.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"target_id": map[string]interface{}{
						"type":        "string",
						"description": "The ID of the target to initialize (from list_targets)",
					},
				},
				"required": []string{"target_id"},
			},
			BackendID: BuiltinBackendID,
		},
		{
			Name:        "stop_target",
			Description: "Stop a running target MCP server and release its resources.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"target_id": map[string]interface{}{
						"type":        "string",
						"description": "The ID of the target to stop",
					},
				},
				"required": []string{"target_id"},
			},
			BackendID: BuiltinBackendID,
		},
		{
			Name:        "setup_database",
			Description: "Setup a database MCP target from an active Apono session. Fetches credentials and automatically initializes the target.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "The ID of the active access session to setup MCP for",
					},
				},
				"required": []string{"session_id"},
			},
			BackendID: BuiltinBackendID,
		},
	}
}

// HandleToolCall handles built-in tool invocations
func (h *BuiltinToolsHandler) HandleToolCall(ctx context.Context, toolName string, arguments json.RawMessage) (interface{}, error) {
	var args map[string]interface{}
	if arguments != nil {
		if err := json.Unmarshal(arguments, &args); err != nil {
			args = make(map[string]interface{})
		}
	}

	switch toolName {
	case "list_targets":
		return h.handleListTargets(ctx)
	case "init_target":
		return h.handleInitTarget(ctx, args)
	case "stop_target":
		return h.handleStopTarget(ctx, args)
	case "setup_database":
		return h.handleSetupDatabase(ctx, args)
	default:
		return nil, fmt.Errorf("unknown built-in tool: %s", toolName)
	}
}

func (h *BuiltinToolsHandler) handleListTargets(ctx context.Context) (interface{}, error) {
	targets, err := h.manager.ListTargets(ctx)
	if err != nil {
		return ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: fmt.Sprintf("error listing targets: %s", err.Error())},
			},
			IsError: true,
		}, nil
	}

	targetsJSON, err := json.MarshalIndent(targets, "", "  ")
	if err != nil {
		return nil, err
	}

	return ToolCallResult{
		Content: []ContentItem{
			{Type: "text", Text: string(targetsJSON)},
		},
	}, nil
}

func (h *BuiltinToolsHandler) handleInitTarget(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	targetID, _ := args["target_id"].(string)
	if targetID == "" {
		return ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: "error: target_id is required"},
			},
			IsError: true,
		}, nil
	}

	utils.McpLogf("[BuiltinTools] Initializing target: %s", targetID)

	if err := h.manager.InitTarget(ctx, targetID); err != nil {
		return ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: fmt.Sprintf("error initializing target: %s", err.Error())},
			},
			IsError: true,
		}, nil
	}

	// Get available tools from the newly initialized target
	tools, err := h.manager.ListToolsForUser(ctx)
	if err != nil {
		tools = []Tool{}
	}

	toolNames := make([]string, len(tools))
	for i, t := range tools {
		toolNames[i] = PrefixToolName(t.BackendID, t.Name)
	}

	responseData := map[string]interface{}{
		"success":         true,
		"target_id":       targetID,
		"available_tools": toolNames,
		"message":         fmt.Sprintf("Target '%s' initialized successfully. Tools are now available with prefix '%s__'.", targetID, targetID),
	}

	responseJSON, _ := json.MarshalIndent(responseData, "", "  ")
	return ToolCallResult{
		Content: []ContentItem{
			{Type: "text", Text: string(responseJSON)},
		},
	}, nil
}

func (h *BuiltinToolsHandler) handleStopTarget(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	targetID, _ := args["target_id"].(string)
	if targetID == "" {
		return ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: "error: target_id is required"},
			},
			IsError: true,
		}, nil
	}

	if err := h.manager.StopTarget(ctx, targetID); err != nil {
		return ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: fmt.Sprintf("error stopping target: %s", err.Error())},
			},
			IsError: true,
		}, nil
	}

	responseData := map[string]interface{}{
		"success":   true,
		"target_id": targetID,
		"message":   "Target stopped successfully",
	}

	responseJSON, _ := json.MarshalIndent(responseData, "", "  ")
	return ToolCallResult{
		Content: []ContentItem{
			{Type: "text", Text: string(responseJSON)},
		},
	}, nil
}

func (h *BuiltinToolsHandler) handleSetupDatabase(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	sessionID, _ := args["session_id"].(string)
	if sessionID == "" {
		return ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: "error: session_id is required"},
			},
			IsError: true,
		}, nil
	}

	apiBaseURL := h.manager.apiBaseURL
	httpClient := h.manager.httpClient
	if apiBaseURL == "" || httpClient == nil {
		return ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: "error: API client not configured for setup_database"},
			},
			IsError: true,
		}, nil
	}

	// List all sessions to find the one matching session_id
	sessionsURL := fmt.Sprintf("%s/api/client/v1/access-sessions", apiBaseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", sessionsURL, nil)
	if err != nil {
		return ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: "error: failed to create request: " + err.Error()},
			},
			IsError: true,
		}, nil
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: "error: failed to list sessions: " + err.Error()},
			},
			IsError: true,
		}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: "error: failed to read sessions response: " + err.Error()},
			},
			IsError: true,
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: fmt.Sprintf("error: failed to list sessions: API returned %d: %s", resp.StatusCode, string(body))},
			},
			IsError: true,
		}, nil
	}

	// Parse sessions list
	type SessionCredentials struct {
		Status   string `json:"status"`
		CanReset bool   `json:"can_reset"`
	}
	type Session struct {
		ID          string `json:"id"`
		Integration struct {
			Name string `json:"name"`
		} `json:"integration"`
		Credentials *SessionCredentials `json:"credentials,omitempty"`
	}
	var sessionsResponse struct {
		Data []Session `json:"data"`
	}
	if err := json.Unmarshal(body, &sessionsResponse); err != nil {
		return ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: "error: failed to parse sessions: " + err.Error()},
			},
			IsError: true,
		}, nil
	}
	sessions := sessionsResponse.Data

	// Find session matching the given ID
	var targetSession *Session
	for i, s := range sessions {
		if s.ID == sessionID {
			targetSession = &sessions[i]
			break
		}
	}

	if targetSession == nil {
		return ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: fmt.Sprintf("error: session not found: %s", sessionID)},
			},
			IsError: true,
		}, nil
	}

	integrationName := targetSession.Integration.Name

	// Check if credentials need to be reset (stale credentials)
	if targetSession.Credentials != nil && targetSession.Credentials.Status != "new" && targetSession.Credentials.CanReset {
		utils.McpLogf("[SetupDatabase] Credentials are stale (status: %s), resetting...", targetSession.Credentials.Status)

		resetURL := fmt.Sprintf("%s/api/client/v1/access-sessions/%s/reset-credentials", apiBaseURL, sessionID)
		resetReq, err := http.NewRequestWithContext(ctx, "POST", resetURL, nil)
		if err != nil {
			return ToolCallResult{
				Content: []ContentItem{
					{Type: "text", Text: "error: failed to create reset request: " + err.Error()},
				},
				IsError: true,
			}, nil
		}

		resetResp, err := httpClient.Do(resetReq)
		if err != nil {
			return ToolCallResult{
				Content: []ContentItem{
					{Type: "text", Text: "error: failed to reset credentials: " + err.Error()},
				},
				IsError: true,
			}, nil
		}
		resetResp.Body.Close()

		if resetResp.StatusCode != http.StatusOK && resetResp.StatusCode != http.StatusNoContent {
			return ToolCallResult{
				Content: []ContentItem{
					{Type: "text", Text: fmt.Sprintf("error: credential reset failed: %d", resetResp.StatusCode)},
				},
				IsError: true,
			}, nil
		}

		// Wait for credentials to become "new"
		maxRetries := 30
		credentialsReady := false
		for i := 0; i < maxRetries; i++ {
			time.Sleep(1 * time.Second)

			checkReq, _ := http.NewRequestWithContext(ctx, "GET", sessionsURL, nil)
			checkResp, err := httpClient.Do(checkReq)
			if err != nil {
				continue
			}

			checkBody, _ := io.ReadAll(checkResp.Body)
			checkResp.Body.Close()

			var updatedResponse struct {
				Data []Session `json:"data"`
			}
			if json.Unmarshal(checkBody, &updatedResponse) == nil {
				for _, s := range updatedResponse.Data {
					if s.ID == sessionID && s.Credentials != nil && s.Credentials.Status == "new" {
						utils.McpLogf("[SetupDatabase] Credentials reset successfully")
						credentialsReady = true
						break
					}
				}
			}
			if credentialsReady {
				break
			}
		}

		if !credentialsReady {
			return ToolCallResult{
				Content: []ContentItem{
					{Type: "text", Text: "error: timeout waiting for credentials to reset"},
				},
				IsError: true,
			}, nil
		}
	}

	// Get session details with credentials
	detailsURL := fmt.Sprintf("%s/api/client/v1/access-sessions/%s/access-details", apiBaseURL, sessionID)
	detailsReq, err := http.NewRequestWithContext(ctx, "GET", detailsURL, nil)
	if err != nil {
		return ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: "error: failed to create details request: " + err.Error()},
			},
			IsError: true,
		}, nil
	}

	detailsResp, err := httpClient.Do(detailsReq)
	if err != nil {
		return ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: "error: failed to get session details: " + err.Error()},
			},
			IsError: true,
		}, nil
	}
	defer detailsResp.Body.Close()

	detailsBody, err := io.ReadAll(detailsResp.Body)
	if err != nil {
		return ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: "error: failed to read details response: " + err.Error()},
			},
			IsError: true,
		}, nil
	}

	if detailsResp.StatusCode != http.StatusOK {
		return ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: fmt.Sprintf("error: failed to get session details: %d: %s", detailsResp.StatusCode, string(detailsBody))},
			},
			IsError: true,
		}, nil
	}

	// Parse session details with credentials
	var sessionDetails struct {
		Json map[string]interface{} `json:"json"`
	}
	if err := json.Unmarshal(detailsBody, &sessionDetails); err != nil {
		return ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: "error: failed to parse session details: " + err.Error()},
			},
			IsError: true,
		}, nil
	}

	// Extract PostgreSQL credentials
	creds := sessionDetails.Json
	host, _ := creds["host"].(string)
	port, _ := creds["port"].(string)
	if portNum, ok := creds["port"].(float64); ok {
		port = fmt.Sprintf("%.0f", portNum)
	}
	dbName, _ := creds["db_name"].(string)
	username, _ := creds["username"].(string)
	password, _ := creds["password"].(string)

	if host == "" || port == "" || dbName == "" || username == "" || password == "" {
		return ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: "error: session is not a PostgreSQL database or missing credentials"},
			},
			IsError: true,
		}, nil
	}

	// Build connection string with URL encoding for special characters
	connectionString := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s",
		url.QueryEscape(username),
		url.QueryEscape(password),
		host, port, dbName)

	// Sanitize integration name for target ID
	targetID := sanitizeName(integrationName)

	// Create postgres target definition
	postgresTarget := targets.TargetDefinition{
		ID:   targetID,
		Name: fmt.Sprintf("Apono: %s", integrationName),
		Type: "postgres",
		Credentials: map[string]string{
			"database_url": connectionString,
		},
	}

	// Write target to targets.yaml
	if h.manager.targetsFilePath != "" {
		fileLoader := targets.NewFileTargetLoader(h.manager.targetsFilePath)
		if err := fileLoader.AddTarget(postgresTarget); err != nil {
			return ToolCallResult{
				Content: []ContentItem{
					{Type: "text", Text: "error: failed to update targets.yaml: " + err.Error()},
				},
				IsError: true,
			}, nil
		}
	}

	utils.McpLogf("[SetupDatabase] Setup database target %s", targetID)

	// Auto-initialize the new target
	if err := h.manager.InitTarget(ctx, targetID); err != nil {
		return ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: "error: target configured but initialization failed: " + err.Error()},
			},
			IsError: true,
		}, nil
	}

	// Get available tools from the newly initialized target
	tools, err := h.manager.ListToolsForUser(ctx)
	if err != nil {
		tools = []Tool{}
	}

	// Build tool names list for the new target only
	var targetTools []string
	for _, t := range tools {
		if t.BackendID == targetID {
			targetTools = append(targetTools, PrefixToolName(t.BackendID, t.Name))
		}
	}

	responseData := map[string]interface{}{
		"success":         true,
		"target_id":       targetID,
		"database":        dbName,
		"host":            host,
		"message":         fmt.Sprintf("Target '%s' configured and initialized!", targetID),
		"available_tools": targetTools,
	}

	responseJSON, _ := json.MarshalIndent(responseData, "", "  ")
	return ToolCallResult{
		Content: []ContentItem{
			{Type: "text", Text: string(responseJSON)},
		},
	}, nil
}

// sanitizeName converts a name to a safe format for use in target IDs
func sanitizeName(name string) string {
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
