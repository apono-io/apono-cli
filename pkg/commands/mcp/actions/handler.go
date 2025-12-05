package actions

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/commands/mcp/tools"
	"github.com/apono-io/apono-cli/pkg/utils"
)

const (
	MCPVersion    = "2024-11-05"
	ServerName    = "apono-cli"
	ServerVersion = "1.0.0"

	ErrorCodeMethodNotFound = -32601
	ErrorCodeInvalidParams  = -32602
)

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type MCPHandler struct {
	toolRegistry *tools.ToolRegistry
	client       *aponoapi.AponoClient
}

func NewMCPHandler(client *aponoapi.AponoClient) *MCPHandler {
	return &MCPHandler{
		toolRegistry: tools.NewToolRegistry(),
		client:       client,
	}
}

func (h *MCPHandler) HandleRequest(ctx context.Context, requestLine string) string {
	var req JSONRPCRequest
	if err := json.Unmarshal([]byte(requestLine), &req); err != nil {
		utils.McpLogf("[Error]: Failed to parse JSON-RPC request: %v", err)
		// Use -1 as a sentinel ID when we can't parse the request
		return h.errorResponse(-1, ErrorCodeInvalidParams, "Invalid JSON-RPC request", err.Error())
	}

	utils.McpLogf("Handling method: %s", req.Method)

	var result interface{}
	var err error

	switch req.Method {
	case "initialize":
		result, err = h.handleInitialize(ctx, req.Params)
	case "initialized":
		// Notification - no response needed
		return ""
	case "tools/list":
		result, err = h.handleToolsList(ctx)
	case "tools/call":
		result, err = h.handleToolsCall(ctx, req.Params)
	case "prompts/list":
		// We don't have prompts, return empty list
		result = map[string]interface{}{"prompts": []interface{}{}}
	case "resources/list":
		// We don't have resources, return empty list
		result = map[string]interface{}{"resources": []interface{}{}}
	case "ping":
		result = map[string]interface{}{}
	default:
		// For unknown methods, check if it's a notification (no ID)
		if req.ID == nil {
			// Notifications don't require a response
			utils.McpLogf("Received notification: %s (no response needed)", req.Method)
			return ""
		}
		return h.errorResponse(req.ID, ErrorCodeMethodNotFound, fmt.Sprintf("Method not found: %s", req.Method), nil)
	}

	if err != nil {
		utils.McpLogf("[Error]: Method %s failed: %v", req.Method, err)
		return h.errorResponse(req.ID, ErrorCodeInternalError, err.Error(), nil)
	}

	return h.successResponse(req.ID, result)
}

func (h *MCPHandler) handleInitialize(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return map[string]interface{}{
		"protocolVersion": MCPVersion,
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    ServerName,
			"version": ServerVersion,
		},
	}, nil
}

func (h *MCPHandler) handleToolsList(ctx context.Context) (interface{}, error) {
	toolsList := h.toolRegistry.ListTools()

	toolSchemas := make([]map[string]interface{}, 0, len(toolsList))
	for _, tool := range toolsList {
		toolSchemas = append(toolSchemas, map[string]interface{}{
			"name":        tool.Name(),
			"description": tool.Description(),
			"inputSchema": tool.InputSchema(),
		})
	}

	return map[string]interface{}{
		"tools": toolSchemas,
	}, nil
}

type ToolsCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

func (h *MCPHandler) handleToolsCall(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var callParams ToolsCallParams
	if err := json.Unmarshal(params, &callParams); err != nil {
		return nil, fmt.Errorf("invalid tool call params: %w", err)
	}

	utils.McpLogf("Calling tool: %s", callParams.Name)

	tool, err := h.toolRegistry.Get(callParams.Name)
	if err != nil {
		return nil, err
	}

	result, err := tool.Execute(ctx, h.client, callParams.Arguments)
	if err != nil {
		return nil, fmt.Errorf("tool execution failed: %w", err)
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": formatToolResult(result),
			},
		},
	}, nil
}

func formatToolResult(result interface{}) string {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", result)
	}
	return string(data)
}

func (h *MCPHandler) successResponse(id interface{}, result interface{}) string {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	data, _ := json.Marshal(resp)
	return string(data)
}

func (h *MCPHandler) errorResponse(id interface{}, code int, message string, data interface{}) string {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	respData, _ := json.Marshal(resp)
	return string(respData)
}
