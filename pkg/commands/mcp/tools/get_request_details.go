package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
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
- access_groups: Detailed breakdown of what resources and permissions you have
  - Each access_group contains:
    - integration_name: Name of the integration (e.g., "local-postgres")
    - integration_type: Type (e.g., "postgresql")
    - resource_names: List of resource names (e.g., ["postgres", "apono", "users_db"])
    - permissions: List of permissions granted (e.g., ["READ", "WRITE", "ADMIN"])
    - resource_count: Total number of resources in this access group
    - has_more: true if there are more than 10 resources (only first 10 names shown)

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
6. Request the correct access via ask_access_assistant with more specific details` + "\n" + FlowDescription + `
(you are here) → step 4: get_request_details`
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
	IntegrationName string   `json:"integration_name"`
	IntegrationType string   `json:"integration_type"`
	ResourceNames   []string `json:"resource_names"`
	Permissions     []string `json:"permissions"`
	ResourceCount   int      `json:"resource_count"`
	HasMore         bool     `json:"has_more"`
	Description     string   `json:"description"`
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

	// Get access units to see actual resources
	accessUnits, err := services.ListAccessRequestAccessUnits(ctx, client, args.RequestID)
	if err != nil {
		fmt.Printf("[DEBUG] Warning: Failed to get access units: %v\n", err)
		accessUnits = []clientapi.AccessUnitClientModel{}
	}

	// Group access units by access group
	accessGroupUnits := make(map[string][]clientapi.AccessUnitClientModel)
	for _, unit := range accessUnits {
		accessGroupUnits[unit.Resource.Integration.Id] = append(accessGroupUnits[unit.Resource.Integration.Id], unit)
	}

	// Parse access groups to extract meaningful information
	var summaryParts []string
	const maxResourcesPerGroup = 10

	for _, accessGroup := range request.AccessGroups {
		groupDetails := AccessGroupDetails{
			IntegrationName: accessGroup.Integration.Name,
			IntegrationType: accessGroup.Integration.Type,
			ResourceNames:   []string{},
			Permissions:     []string{},
		}

		// Get units for this access group
		units := accessGroupUnits[accessGroup.Integration.Id]
		groupDetails.ResourceCount = len(units)
		groupDetails.HasMore = len(units) > maxResourcesPerGroup

		// Collect unique resource names and permissions
		resourceNameSet := make(map[string]bool)
		permissionSet := make(map[string]bool)

		for i, unit := range units {
			if i >= maxResourcesPerGroup {
				break
			}
			resourceNameSet[unit.Resource.Name] = true
			permissionSet[unit.Permission.Name] = true
		}

		// Convert sets to slices
		for name := range resourceNameSet {
			groupDetails.ResourceNames = append(groupDetails.ResourceNames, name)
		}
		for perm := range permissionSet {
			groupDetails.Permissions = append(groupDetails.Permissions, perm)
		}

		// Build description
		description := fmt.Sprintf("Access to %s (%s)", accessGroup.Integration.Name, accessGroup.Integration.Type)
		if groupDetails.ResourceCount > 0 {
			description = fmt.Sprintf("%s - %d resource(s)", description, groupDetails.ResourceCount)
			if groupDetails.HasMore {
				description = fmt.Sprintf("%s (showing first %d)", description, maxResourcesPerGroup)
			}
		} else {
			description = fmt.Sprintf("%s - All resources", description)
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

	// Determine next_step based on request status
	var nextStep string
	statusUpper := strings.ToUpper(response.Status)
	if strings.Contains(statusUpper, "APPROVED") || strings.Contains(statusUpper, "ACTIVE") {
		nextStep = "Access granted. Database tools are now available in your tool list. Use list_targets to see connected targets and their tools."
	} else {
		nextStep = "Request is still pending. Call get_request_details again to check status."
	}

	return map[string]interface{}{
		"request":   response,
		"next_step": nextStep,
	}, nil
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
