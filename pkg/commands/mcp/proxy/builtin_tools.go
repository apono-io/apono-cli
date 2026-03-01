package proxy

import (
	"context"
	"encoding/json"
	"fmt"

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
	case "setup_database":
		return h.handleSetupDatabase(ctx, args)
	default:
		return nil, fmt.Errorf("unknown built-in tool: %s", toolName)
	}
}

func (h *BuiltinToolsHandler) handleListTargets(ctx context.Context) (interface{}, error) {
	targetList, err := h.manager.ListTargets(ctx)
	if err != nil {
		return nil, err
	}

	connectedTargets := make([]map[string]interface{}, 0)
	pendingRequests := make([]map[string]interface{}, 0)

	for _, t := range targetList {
		if t.Initialized {
			toolNames := make([]string, 0)
			if inst := h.manager.getInstance(t.ID); inst != nil {
				tools, err := inst.Backend.ListTools(ctx)
				if err == nil {
					for _, tool := range tools {
						toolNames = append(toolNames, PrefixToolName(t.ID, tool.Name))
					}
				}
			}
			connectedTargets = append(connectedTargets, map[string]interface{}{
				"name":            t.Name,
				"type":            t.Type,
				"status":          "connected",
				"available_tools": toolNames,
			})
		} else if t.Status == targets.TargetStatusNeedsAccess {
			pendingRequests = append(pendingRequests, map[string]interface{}{
				"name":   t.Name,
				"status": string(t.Status),
			})
		}
	}

	responseJSON, err := json.MarshalIndent(map[string]interface{}{
		"connected_targets": connectedTargets,
		"pending_requests":  pendingRequests,
		"next_step":         "You can now use the tools listed above directly.",
	}, "", "  ")
	if err != nil {
		return nil, err
	}

	return ToolCallResult{
		Content: []ContentItem{
			{Type: "text", Text: string(responseJSON)},
		},
	}, nil
}

func (h *BuiltinToolsHandler) handleSetupDatabase(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	sessionID, ok := args["session_id"].(string)
	if !ok || sessionID == "" {
		return &ToolCallResult{
			Content: []ContentItem{{Type: "text", Text: "Error: session_id is required"}},
			IsError: true,
		}, nil
	}

	utils.McpLogf("[SetupDatabase] Ensuring access for session %s", sessionID)

	// Trigger EnsureAccess which will make the session "ready"
	// The session watcher will then detect it and auto-spawn the backend
	err := h.manager.TargetSource().EnsureAccess(ctx, sessionID)
	if err != nil {
		return &ToolCallResult{
			Content: []ContentItem{{Type: "text", Text: fmt.Sprintf("Failed to ensure access: %v", err)}},
			IsError: true,
		}, nil
	}

	resultJSON, _ := json.Marshal(map[string]interface{}{
		"status":    "success",
		"message":   "Access ensured. The MCP server will be automatically spawned when the session becomes active.",
		"next_step": "Use list_targets to check when the MCP server is ready and see available tools.",
	})

	return &ToolCallResult{
		Content: []ContentItem{{Type: "text", Text: string(resultJSON)}},
	}, nil
}
