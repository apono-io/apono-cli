package proxy

import (
	"context"
	"encoding/json"
)

// Backend represents an MCP backend server (subprocess)
type Backend interface {
	ID() string
	Name() string
	Type() string
	Send(ctx context.Context, request []byte) ([]byte, error)
	Initialize(ctx context.Context) error
	Close() error
	Health(ctx context.Context) error
	IsReady() bool
	ListTools(ctx context.Context) ([]Tool, error)
}

// Tool represents an MCP tool from a backend
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
	BackendID   string                 `json:"backend_id,omitempty"`
}

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      interface{}      `json:"id"`
	Result  *json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError    `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ToolsListResult represents the result of tools/list
type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

// ContentItem represents a content item in tool responses
type ContentItem struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Data     string `json:"data,omitempty"`
}

// ToolCallResult represents the result of tools/call
type ToolCallResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// Error codes
const (
	ErrorCodeMethodNotFound  = -32601
	ErrorCodeInvalidParams   = -32602
	ErrorCodeInternalError   = -32603
	ErrorCodeBlockedByPolicy = -32000
	ErrorCodeBackendNotFound = -32001
	ErrorCodeBackendError    = -32002
)

// NewJSONRPCRequest creates a new JSON-RPC request
func NewJSONRPCRequest(id interface{}, method string, params interface{}) *JSONRPCRequest {
	return &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
}

// CreateErrorResponse creates a JSON-RPC error response
func CreateErrorResponse(id interface{}, code int, message string, data interface{}) []byte {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	b, _ := json.Marshal(resp)
	return b
}

// DynamicToolSchema represents a tool from the proxy layer for listing
type DynamicToolSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}
