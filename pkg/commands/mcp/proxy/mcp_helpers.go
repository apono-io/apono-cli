package proxy

import (
	"context"
	"encoding/json"
	"fmt"
)

// Sender is an interface for backends that can send JSON-RPC requests
type Sender interface {
	Send(ctx context.Context, request []byte) ([]byte, error)
}

// MCPClient provides common MCP operations using a Sender
type MCPClient struct {
	sender    Sender
	backendID string
	getNextID func() interface{}
}

// NewMCPClient creates a new MCP client wrapper
func NewMCPClient(sender Sender, backendID string, getNextID func() interface{}) *MCPClient {
	return &MCPClient{
		sender:    sender,
		backendID: backendID,
		getNextID: getNextID,
	}
}

// ListTools fetches tools from the backend and sets BackendID
func (c *MCPClient) ListTools(ctx context.Context) ([]Tool, error) {
	req := NewJSONRPCRequest(c.getNextID(), "tools/list", nil)
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	respBytes, err := c.sender.Send(ctx, reqBytes)
	if err != nil {
		return nil, err
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("tools/list error: %s", resp.Error.Message)
	}

	if resp.Result == nil {
		return []Tool{}, nil
	}

	var result ToolsListResult
	if err := json.Unmarshal(*resp.Result, &result); err != nil {
		return nil, err
	}

	for i := range result.Tools {
		result.Tools[i].BackendID = c.backendID
	}

	return result.Tools, nil
}

// ExtractTargetFromArgs extracts and removes a parameter from tool arguments
func ExtractTargetFromArgs(args map[string]interface{}, paramName string, validValues []string) (string, map[string]interface{}, error) {
	value, ok := args[paramName].(string)
	if !ok || value == "" {
		return "", nil, fmt.Errorf("missing required '%s' parameter (valid: %v)", paramName, validValues)
	}

	found := false
	for _, v := range validValues {
		if v == value {
			found = true
			break
		}
	}
	if !found {
		return "", nil, fmt.Errorf("unknown %s: %s (valid: %v)", paramName, value, validValues)
	}

	// Remove parameter from args
	modifiedArgs := make(map[string]interface{})
	for k, v := range args {
		if k != paramName {
			modifiedArgs[k] = v
		}
	}

	return value, modifiedArgs, nil
}

// RebuildToolCallRequest creates a modified tools/call request with new arguments
func RebuildToolCallRequest(reqID interface{}, toolName string, args map[string]interface{}) ([]byte, error) {
	modifiedReq := NewJSONRPCRequest(reqID, "tools/call", map[string]interface{}{
		"name":      toolName,
		"arguments": args,
	})
	return json.Marshal(modifiedReq)
}

// InjectEnumParameter adds an enum parameter to a tool's input schema
func InjectEnumParameter(tool Tool, paramName, description string, enumValues []string) Tool {
	if tool.InputSchema == nil {
		tool.InputSchema = make(map[string]interface{})
	}

	if tool.InputSchema["type"] == nil {
		tool.InputSchema["type"] = "object"
	}

	properties, _ := tool.InputSchema["properties"].(map[string]interface{})
	if properties == nil {
		properties = make(map[string]interface{})
		tool.InputSchema["properties"] = properties
	}

	properties[paramName] = map[string]interface{}{
		"type":        "string",
		"description": description,
		"enum":        enumValues,
	}

	required, _ := tool.InputSchema["required"].([]interface{})
	if required == nil {
		required = []interface{}{}
	}

	hasRequired := false
	for _, r := range required {
		if r == paramName {
			hasRequired = true
			break
		}
	}
	if !hasRequired {
		required = append(required, paramName)
		tool.InputSchema["required"] = required
	}

	return tool
}
