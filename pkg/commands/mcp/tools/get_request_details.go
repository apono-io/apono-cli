package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/services"
)

type GetRequestDetailsTool struct{}

func (t *GetRequestDetailsTool) Name() string {
	return "get_request_details"
}

func (t *GetRequestDetailsTool) Description() string {
	return `Get detailed information about a specific access request.

⭐ WHEN TO USE:
1. After creating an access request to verify what access was granted
2. When you get permission errors and want to check if your active access includes the right resource/database
3. To understand what databases/resources are included in your current access
4. To verify if you requested access to the correct resource

WHAT IT RETURNS:
- request_id: The access request ID
- status: Current status (Active, Pending, Granting, etc.)
- created_at: When the request was created
- expires_at: When the access will expire
- integrations: List of integrations this request grants access to
- access_groups: Detailed breakdown of what resources and permissions you have
  - Each access_group contains:
    - integration_name: Name of the integration (e.g., "local-postgres")
    - integration_type: Type (e.g., "postgresql")
    - resources: Specific resources granted (databases, schemas, tables)
    - permissions: What you can do (READ, WRITE, ADMIN, etc.)

🎯 USE THIS TO VERIFY YOUR ACCESS:
After requesting access and still getting errors, use this tool to check:
- "Does my access include the database I'm trying to use?"
- "Did I get access to 'postgres' database when I needed 'apono' database?"
- "What specific resources am I allowed to access?"

EXAMPLE WORKFLOW:
1. You request "database access"
2. Access is granted (request becomes Active)
3. You try to query a table and get permission denied
4. Call this tool with your request_id to see what was actually granted
5. Discover you got access to wrong database or wrong permissions
6. Request the correct access via ask_access_assistant with more specific details`
}

func (t *GetRequestDetailsTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"request_id": map[string]interface{}{
				"type":        "string",
				"description": "The access request ID to get details for (e.g., 'AR-00001')",
			},
		},
		"required": []string{"request_id"},
	}
}

type GetRequestDetailsArgs struct {
	RequestID string `json:"request_id"`
}

type RequestDetailsResponse struct {
	RequestID     string               `json:"request_id"`
	Status        string               `json:"status"`
	CreatedAt     string               `json:"created_at"`
	ExpiresAt     string               `json:"expires_at,omitempty"`
	Justification string               `json:"justification,omitempty"`
	AccessGroups  []AccessGroupDetails `json:"access_groups"`
	Summary       string               `json:"summary"`
}

type AccessGroupDetails struct {
	IntegrationName string            `json:"integration_name"`
	IntegrationType string            `json:"integration_type"`
	Resources       []ResourceDetails `json:"resources,omitempty"`
	Description     string            `json:"description"`
}

type ResourceDetails struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Permissions []string `json:"permissions,omitempty"`
}

func (t *GetRequestDetailsTool) Execute(ctx context.Context, client *aponoapi.AponoClient, arguments json.RawMessage) (interface{}, error) {
	var args GetRequestDetailsArgs
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.RequestID == "" {
		return nil, fmt.Errorf("request_id is required")
	}

	fmt.Printf("[DEBUG] Getting details for request: %s\n", args.RequestID)

	// Get the request details
	request, err := services.GetRequestByID(ctx, client, args.RequestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get request details: %w", err)
	}

	// Build the response
	response := RequestDetailsResponse{
		RequestID: request.Id,
		Status:    request.Status.Status,
		CreatedAt: fmt.Sprintf("%d", request.CreationTime),
	}

	if request.RevocationTime.IsSet() {
		response.ExpiresAt = fmt.Sprintf("%d", *request.RevocationTime.Get())
	}

	if request.Justification.IsSet() {
		response.Justification = *request.Justification.Get()
	}

	// Parse access groups to extract meaningful information
	var summaryParts []string
	for _, accessGroup := range request.AccessGroups {
		groupDetails := AccessGroupDetails{
			IntegrationName: accessGroup.Integration.Name,
			IntegrationType: accessGroup.Integration.Type,
		}

		// Build description based on what's in the access group
		description := fmt.Sprintf("Access to %s (%s)", accessGroup.Integration.Name, accessGroup.Integration.Type)

		// Add resource types if available
		if len(accessGroup.ResourceTypes) > 0 {
			var resourceTypeNames []string
			for _, rt := range accessGroup.ResourceTypes {
				resourceTypeNames = append(resourceTypeNames, rt.Name)
			}
			description = fmt.Sprintf("%s - Resource types: %s", description, joinStrings(resourceTypeNames, ", "))
		}

		groupDetails.Description = description
		response.AccessGroups = append(response.AccessGroups, groupDetails)

		summaryParts = append(summaryParts, fmt.Sprintf("%s (%s)", accessGroup.Integration.Name, accessGroup.Integration.Type))
	}

	if len(summaryParts) > 0 {
		response.Summary = fmt.Sprintf("This request grants access to: %s", joinStrings(summaryParts, ", "))
	} else {
		response.Summary = "This request has no active access groups"
	}

	fmt.Printf("[DEBUG] Request status: %s, Access groups: %d\n", response.Status, len(response.AccessGroups))

	return response, nil
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
