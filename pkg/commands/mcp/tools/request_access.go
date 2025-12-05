package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/services"
)

type RequestAccessTool struct{}

func (t *RequestAccessTool) Name() string {
	return "request_access"
}

func (t *RequestAccessTool) Description() string {
	return `⚠️ DEPRECATED - DO NOT USE THIS TOOL ⚠️

This tool is obsolete and only lists bundles which are often empty.

INSTEAD, ALWAYS USE:
1. ask_access_assistant - Understands what access you need from natural language, works for ALL resource types
2. create_access_request - Actually submits the access request after the assistant builds it

The assistant handles databases, Kubernetes, clouds, SaaS tools, and any other resource type.
It will ask clarifying questions and build the correct request for you.`
}

func (t *RequestAccessTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"integration_id": map[string]interface{}{
				"type":        "string",
				"description": "The ID of the integration to request access to",
			},
			"permission_level": map[string]interface{}{
				"type":        "string",
				"description": "Optional: The permission level to request (e.g., 'read', 'write', 'admin'). If not specified, will request default access.",
			},
			"justification": map[string]interface{}{
				"type":        "string",
				"description": "Optional: Justification for the access request",
			},
			"duration_minutes": map[string]interface{}{
				"type":        "integer",
				"description": "Optional: Duration of access in minutes (default: 60)",
			},
		},
		"required": []string{"integration_id"},
	}
}

type RequestAccessInput struct {
	IntegrationID   string `json:"integration_id"`
	PermissionLevel string `json:"permission_level,omitempty"`
	Justification   string `json:"justification,omitempty"`
	DurationMinutes int32  `json:"duration_minutes,omitempty"`
}

func (t *RequestAccessTool) Execute(ctx context.Context, client *aponoapi.AponoClient, arguments json.RawMessage) (interface{}, error) {
	var input RequestAccessInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if input.IntegrationID == "" {
		return nil, fmt.Errorf("integration_id is required")
	}

	// Get available bundles
	bundles, err := services.ListBundles(ctx, client, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list bundles: %w", err)
	}

	// Convert bundles to simple format
	bundleInfo := make([]map[string]interface{}, 0)
	for _, bundle := range bundles {
		bundleInfo = append(bundleInfo, map[string]interface{}{
			"id":   bundle.Id,
			"name": bundle.Name,
		})
	}

	return map[string]interface{}{
		"integration_id":    input.IntegrationID,
		"available_bundles": bundleInfo,
		"count":             len(bundleInfo),
		"message":           fmt.Sprintf("Found %d available bundles. To request access, please use the CLI command: apono requests new --integration-id %s", len(bundleInfo), input.IntegrationID),
	}, nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr))
}
