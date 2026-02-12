package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

// MCPTool represents a tool that can be called via MCP
type MCPTool interface {
	Name() string
	Description() string
	InputSchema() map[string]interface{}
	Execute(ctx context.Context, client *aponoapi.AponoClient, arguments json.RawMessage) (interface{}, error)
}

// ToolRegistry manages all available MCP tools
type ToolRegistry struct {
	tools map[string]MCPTool
}

// NewToolRegistry creates a new tool registry with all available tools
func NewToolRegistry() *ToolRegistry {
	registry := &ToolRegistry{
		tools: make(map[string]MCPTool),
	}

	// Register all tools
	registry.Register(&ListAvailableResourcesTool{})
	// SetupDatabaseMCPTool is deprecated - keeping code but not registering
	// registry.Register(&SetupDatabaseMCPTool{})
	// SetupDatabaseMCPV2Tool is now implemented directly in proxy - not exposing to avoid duplicates
	// registry.Register(&SetupDatabaseMCPV2Tool{})
	registry.Register(&AskAccessAssistantTool{})
	registry.Register(&CreateAccessRequestTool{})
	registry.Register(&GetRequestDetailsTool{})
	registry.Register(&ListResourcesFilteredTool{})

	return registry
}

// Register adds a tool to the registry
func (r *ToolRegistry) Register(tool MCPTool) {
	r.tools[tool.Name()] = tool
}

// Get retrieves a tool by name
func (r *ToolRegistry) Get(name string) (MCPTool, error) {
	tool, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return tool, nil
}

// ListTools returns all available tools
func (r *ToolRegistry) ListTools() []MCPTool {
	tools := make([]MCPTool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}
