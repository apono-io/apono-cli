package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/services"
)

type ListResourcesFilteredTool struct{}

func (t *ListResourcesFilteredTool) Name() string {
	return "list_resources_filtered"
}

func (t *ListResourcesFilteredTool) Description() string {
	return `List available resources filtered by type and/or integration.

⭐ WHEN TO USE:
1. When you need to explore what specific resources are available (e.g., "show me all PostgreSQL databases")
2. Before requesting access, to find the exact integration/resource you need
3. When assistant suggests access but you want to verify the right resource exists
4. To discover what resources of a specific type are available

FILTERS:
- integration_type: Filter by resource type (postgresql, mysql, kubernetes, aws, gcp, etc.)
- integration_name: Filter by integration name (e.g., "local-postgres", "prod-db")
- Both filters are optional - use one or both or neither (lists everything)

WHAT IT RETURNS:
- List of integrations matching your filters
- For each integration:
  - integration_id: Use this when creating access requests
  - integration_name: Human-readable name
  - type: Resource type (postgresql, kubernetes, etc.)
  - type_display_name: Friendly type name
  - has_active_session: Whether you currently have access
  - session_id: If you have access, the session ID

🎯 COMMON USE CASES:

1. Find all PostgreSQL databases:
   {"integration_type": "postgresql"}

2. Find specific integration by name:
   {"integration_name": "local-postgres"}

3. Check if specific database exists:
   {"integration_name": "apono-db", "integration_type": "postgresql"}

4. List all available resource types:
   {} (no filters - lists everything, you can see all types)

💡 WORKFLOW EXAMPLE:
1. User: "I need access to the apono database"
2. You: Call list_resources_filtered with {"integration_name": "apono"}
3. See results, find "apono-db" postgresql integration
4. You: Call ask_access_assistant with specific details: "I need write access to the apono database on integration apono-db (ID: xxx)"
5. Create access request with the exact integration_id

This helps you find the RIGHT resource before requesting access!`
}

func (t *ListResourcesFilteredTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"integration_type": map[string]interface{}{
				"type":        "string",
				"description": "Filter by integration type (e.g., 'postgresql', 'mysql', 'kubernetes', 'aws'). Case-insensitive partial match.",
			},
			"integration_name": map[string]interface{}{
				"type":        "string",
				"description": "Filter by integration name (e.g., 'local-postgres', 'prod'). Case-insensitive partial match.",
			},
		},
	}
}

type ListResourcesFilteredArgs struct {
	IntegrationType string `json:"integration_type,omitempty"`
	IntegrationName string `json:"integration_name,omitempty"`
}

type FilteredResourceInfo struct {
	IntegrationID    string `json:"integration_id"`
	IntegrationName  string `json:"integration_name"`
	Type             string `json:"type"`
	TypeDisplayName  string `json:"type_display_name"`
	HasActiveSession bool   `json:"has_active_session"`
	SessionID        string `json:"session_id,omitempty"`
}

type FilteredResourcesResponse struct {
	Resources []FilteredResourceInfo `json:"resources"`
	Total     int                    `json:"total"`
	Summary   string                 `json:"summary"`
}

func (t *ListResourcesFilteredTool) Execute(ctx context.Context, client *aponoapi.AponoClient, arguments json.RawMessage) (interface{}, error) {
	var args ListResourcesFilteredArgs
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	fmt.Printf("[DEBUG] Filtering resources - type: %q, name: %q\n", args.IntegrationType, args.IntegrationName)

	// Get all integrations
	integrations, err := services.ListIntegrations(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to list integrations: %w", err)
	}

	// Get all active sessions to check which resources have active access
	sessions, err := services.ListAccessSessions(ctx, client, []string{}, []string{}, []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	// Create a map of integration ID -> session ID
	sessionMap := make(map[string]string)
	for _, session := range sessions {
		sessionMap[session.Integration.Id] = session.Id
	}

	// Filter and build results
	var resources []FilteredResourceInfo
	for _, integration := range integrations {
		// Apply filters
		if args.IntegrationType != "" {
			if !strings.Contains(strings.ToLower(integration.Type), strings.ToLower(args.IntegrationType)) {
				continue
			}
		}

		if args.IntegrationName != "" {
			if !strings.Contains(strings.ToLower(integration.Name), strings.ToLower(args.IntegrationName)) {
				continue
			}
		}

		// Build resource info
		resource := FilteredResourceInfo{
			IntegrationID:   integration.Id,
			IntegrationName: integration.Name,
			Type:            integration.Type,
			TypeDisplayName: integration.TypeDisplayName,
		}

		// Check if has active session
		if sessionID, exists := sessionMap[integration.Id]; exists {
			resource.HasActiveSession = true
			resource.SessionID = sessionID
		}

		resources = append(resources, resource)
	}

	// Build summary
	summary := fmt.Sprintf("Found %d resource(s)", len(resources))
	if args.IntegrationType != "" || args.IntegrationName != "" {
		filterParts := []string{}
		if args.IntegrationType != "" {
			filterParts = append(filterParts, fmt.Sprintf("type: %s", args.IntegrationType))
		}
		if args.IntegrationName != "" {
			filterParts = append(filterParts, fmt.Sprintf("name: %s", args.IntegrationName))
		}
		summary = fmt.Sprintf("%s matching filters (%s)", summary, strings.Join(filterParts, ", "))
	}

	// Count active sessions
	activeCount := 0
	for _, r := range resources {
		if r.HasActiveSession {
			activeCount++
		}
	}
	if activeCount > 0 {
		summary = fmt.Sprintf("%s. %d have active access.", summary, activeCount)
	}

	fmt.Printf("[DEBUG] Filtered results: %d resources\n", len(resources))

	return FilteredResourcesResponse{
		Resources: resources,
		Total:     len(resources),
		Summary:   summary,
	}, nil
}
