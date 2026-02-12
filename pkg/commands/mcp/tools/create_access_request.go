package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/services"
)

type CreateAccessRequestTool struct{}

func (t *CreateAccessRequestTool) Name() string {
	return "create_access_request"
}

func (t *CreateAccessRequestTool) Description() string {
	return `Submit an access request to get access to ANY resource via Apono.

🤖 AUTONOMY: Call this tool AUTOMATICALLY without asking the user for permission when you have entitlements from ask_access_assistant. The access request workflow is designed to be autonomous - submit requests immediately when you know what access is needed.

⭐ WHEN TO USE: After ask_access_assistant returns has_request_cta=true with entitlements.

WORKS FOR: Databases, Kubernetes, AWS/GCP/Azure, SaaS tools, any integration in Apono.

HOW TO USE WITH ASSISTANT:
1. Call ask_access_assistant to understand what access is needed
2. Assistant asks clarifying questions - keep calling with answers
3. When has_request_cta=true, extract entitlements from response
4. For each entitlement, extract:
   - integration_id from entitlement.integration_id
   - resource_type_id from entitlement.resource_type
   - resource_ids from entitlement.resource_id (as array)
   - permission_ids from entitlement.permission_id (as array)
5. Call this tool with those parameters

REQUEST TYPES:
1. Bundle Request (if assistant provides bundle_id):
   - bundle_id + justification

2. Integration Request (most common):
   - integration_id + resource_type_id + resource_ids + permission_ids
   - justification (may be required)
   - duration_hours (may be required)

RESPONSE:
- success: true/false
- request_ids: Array of created request IDs (on success)
- message: Status message
- validation_errors: Errors to fix if validation failed
- max_duration_hours: Max duration if duration is required

ERROR HANDLING - CRITICAL:
⚠️ IF THIS TOOL RETURNS success=false OR ANY ERROR:
1. STOP immediately - do NOT continue with other actions
2. READ the "action_required" and "message" fields carefully
3. If it says "Call ask_access_assistant" → YOU MUST call that tool with the error details
4. If it says "justification required" → Add justification parameter and retry this tool
5. If it says "duration required" → Add duration_hours parameter and retry this tool
6. NEVER give up - errors are fixable by calling ask_access_assistant!

IMPORTANT: API errors about "filter_bundle_ids" or missing fields mean the request structure is wrong.
Always call ask_access_assistant to get help building the correct request structure.

AFTER SUCCESS:
- Access request is submitted and will be auto-approved (or require approval)
- The response includes request_ids array (e.g., ["AR-00123"])
- IMPORTANT: Call get_request_details with the request_id to verify what was granted:
  - Check resource_names to ensure you got access to the RIGHT resources
  - Check permissions to ensure you have the right level of access (READ vs WRITE)
  - If you got the wrong resource, request again with more specific details
- Call list_available_resources again - status will change to "ready" when approved
- You can then use the resource (query database, access cluster, etc.)

⚠️ VERIFY YOUR ACCESS:
After requesting access, if you still get permission errors:
1. Call get_request_details to see EXACTLY what resources you were granted
2. Check if the resource_names list includes what you need
3. If not, you may have requested the wrong resource - call ask_access_assistant again with more specific details`

}

func (t *CreateAccessRequestTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"bundle_id": map[string]interface{}{
				"type":        "string",
				"description": "Bundle ID for bundle-based requests (alternative to integration request)",
			},
			"integration_id": map[string]interface{}{
				"type":        "string",
				"description": "Integration ID for integration-based requests (required if not using bundle_id)",
			},
			"resource_type_id": map[string]interface{}{
				"type":        "string",
				"description": "Resource type ID (required for integration requests)",
			},
			"resource_ids": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "List of resource IDs to request access to",
			},
			"permission_ids": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "List of permission IDs to request",
			},
			"justification": map[string]interface{}{
				"type":        "string",
				"description": "Justification for the access request (may be required based on policy)",
			},
			"duration_hours": map[string]interface{}{
				"type":        "number",
				"description": "Duration in hours for the access (may be required based on policy)",
			},
		},
	}
}

type CreateAccessRequestArgs struct {
	BundleID       string   `json:"bundle_id,omitempty"`
	IntegrationID  string   `json:"integration_id,omitempty"`
	ResourceTypeID string   `json:"resource_type_id,omitempty"`
	ResourceIDs    []string `json:"resource_ids,omitempty"`
	PermissionIDs  []string `json:"permission_ids,omitempty"`
	Justification  string   `json:"justification,omitempty"`
	DurationHours  float64  `json:"duration_hours,omitempty"`
}

func (t *CreateAccessRequestTool) Execute(ctx context.Context, client *aponoapi.AponoClient, arguments json.RawMessage) (interface{}, error) {
	var args CreateAccessRequestArgs
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Validate inputs
	if args.BundleID == "" && args.IntegrationID == "" {
		return nil, fmt.Errorf("either bundle_id or integration_id must be provided")
	}

	if args.IntegrationID != "" && args.ResourceTypeID == "" {
		return nil, fmt.Errorf("resource_type_id is required for integration requests")
	}

	// Build the request model - use nil instead of empty arrays to avoid API validation issues
	var filterIntegrationIds []string
	var filterBundleIds []string
	var filterResourceTypeIds []string
	var filterResourceIds []string
	var filterResources []clientapi.ResourceFilter
	var filterPermissionIds []string
	var filterAccessUnitIds []string

	if args.BundleID != "" {
		// Bundle request - only set bundle IDs, use empty arrays for others (API doesn't accept null)
		filterBundleIds = []string{args.BundleID}
		filterIntegrationIds = []string{}
		filterResourceTypeIds = []string{}
		filterResources = []clientapi.ResourceFilter{}
		filterResourceIds = []string{}
		filterPermissionIds = []string{}
		filterAccessUnitIds = []string{}
	} else {
		// Integration request - use empty arrays instead of nil (API doesn't accept null)
		filterIntegrationIds = []string{args.IntegrationID}
		filterResourceTypeIds = []string{args.ResourceTypeID}
		filterBundleIds = []string{} // Empty array, not nil

		if len(args.ResourceIDs) > 0 {
			filterResources = make([]clientapi.ResourceFilter, 0, len(args.ResourceIDs))
			for _, resourceID := range args.ResourceIDs {
				filter := clientapi.NewResourceFilter(clientapi.RESOURCEFILTERTYPE_ID, resourceID)
				filterResources = append(filterResources, *filter)
			}
			filterResourceIds = []string{} // Empty array, not nil
		} else {
			filterResources = []clientapi.ResourceFilter{}
			filterResourceIds = []string{}
		}

		if len(args.PermissionIDs) > 0 {
			filterPermissionIds = args.PermissionIDs
		} else {
			filterPermissionIds = []string{}
		}

		filterAccessUnitIds = []string{} // Empty array, not nil
	}

	request := clientapi.NewCreateAccessRequestClientModel(
		filterIntegrationIds,
		filterBundleIds,
		filterResourceTypeIds,
		filterResourceIds,
		filterResources,
		filterPermissionIds,
		filterAccessUnitIds,
	)

	if args.Justification != "" {
		request.Justification.Set(&args.Justification)
	}

	if args.DurationHours > 0 {
		durationSec := int32(args.DurationHours * 3600)
		request.DurationInSec.Set(&durationSec)
	}

	// Debug: Show what we're about to send
	requestJSON, _ := json.MarshalIndent(request, "", "  ")
	fmt.Printf("[DEBUG] Request payload:\n%s\n", string(requestJSON))

	// Dry run to validate
	config := client.ClientAPI.GetConfig()
	var baseURL string
	if len(config.Servers) > 0 && config.Servers[0].URL != "" {
		baseURL = config.Servers[0].URL
	} else {
		baseURL = fmt.Sprintf("%s://%s", config.Scheme, config.Host)
	}
	fmt.Printf("[DEBUG] Calling dry-run API: POST %s/api/client/v1/access-requests/dry-run\n", baseURL)

	dryRunResult, err := services.DryRunRequest(ctx, client, request)
	if err != nil {
		// API returned an error - FAIL THE TOOL so Cursor sees this as an error
		errorMsg := err.Error()
		fmt.Printf("[DEBUG] Dry-run API error: %s\n", errorMsg)

		// Return an actual error to make this tool call fail
		return nil, fmt.Errorf("❌ Access request failed! Next step: Call ask_access_assistant and explain: 'I tried to create an access request but got this error: %s'. The assistant will help build a correct request", errorMsg)
	}

	fmt.Printf("[DEBUG] Dry-run result: Status=%s, Errors=%d\n", dryRunResult.Status, len(dryRunResult.Errors))

	// Check if validation passed
	if dryRunResult.Status != "ok" && len(dryRunResult.Errors) > 0 {
		errorMsgs := make([]string, 0, len(dryRunResult.Errors))
		for _, err := range dryRunResult.Errors {
			if err.Message.IsSet() && err.Message.Get() != nil {
				errorMsgs = append(errorMsgs, *err.Message.Get())
			}
		}
		// Return actual error so Cursor sees this as a failure
		return nil, fmt.Errorf("❌ Request validation failed! Errors: %v. Next step: Call ask_access_assistant and explain these validation errors to get help fixing the request", errorMsgs)
	}

	// Check if justification is required but not provided
	if !services.IsJustificationOptionalForRequest(dryRunResult) && args.Justification == "" {
		// This is an easy fix - return structured response (not error) so Cursor can retry easily
		return map[string]interface{}{
			"success":         false,
			"message":         "❌ MISSING REQUIRED FIELD: justification",
			"action_required": "IMMEDIATELY retry create_access_request with ALL the same parameters PLUS add justification field",
			"required_field":  "justification",
			"example":         `"justification": "Need write access to insert user data"`,
		}, nil
	}

	// Check if duration is required but not provided
	if services.IsDurationRequiredForRequest(dryRunResult) && args.DurationHours == 0 {
		maxDuration := services.GetMaximumRequestDuration(dryRunResult)
		maxHours := maxDuration.Hours()
		// This is an easy fix - return structured response (not error) so Cursor can retry easily
		return map[string]interface{}{
			"success":            false,
			"message":            fmt.Sprintf("❌ MISSING REQUIRED FIELD: duration_hours (max: %.1f hours)", maxHours),
			"action_required":    "IMMEDIATELY retry create_access_request with ALL the same parameters PLUS add duration_hours field",
			"required_field":     "duration_hours",
			"max_duration_hours": maxHours,
			"example":            fmt.Sprintf(`"duration_hours": %.1f`, maxHours),
		}, nil
	}

	// Create the request
	fmt.Printf("[DEBUG] Calling create request API: POST %s/api/client/v1/access-requests\n", baseURL)

	createdRequest, httpResp, err := client.ClientAPI.AccessRequestsAPI.CreateUserAccessRequest(ctx).
		CreateAccessRequestClientModel(*request).
		Execute()

	if httpResp != nil {
		fmt.Printf("[DEBUG] Create request response status: %d\n", httpResp.StatusCode)
	}

	if err != nil {
		if httpResp != nil && httpResp.StatusCode >= 400 {
			return nil, fmt.Errorf("failed to create access request (status %d): %w", httpResp.StatusCode, err)
		}
		return nil, fmt.Errorf("failed to create access request: %w", err)
	}

	fmt.Printf("[DEBUG] Request created successfully! IDs: %v\n", createdRequest.RequestIds)

	// Format the response
	result := map[string]interface{}{
		"success":     true,
		"request_ids": createdRequest.RequestIds,
		"message":     fmt.Sprintf("Access request created successfully! Request IDs: %v", createdRequest.RequestIds),
	}

	if createdRequest.Message.IsSet() && createdRequest.Message.Get() != nil {
		result["server_message"] = *createdRequest.Message.Get()
	}

	// Provide next steps
	result["next_steps"] = "Request submitted. Check request status with `apono requests list` or wait for approval notification."

	return result, nil
}
