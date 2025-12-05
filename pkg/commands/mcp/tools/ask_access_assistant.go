package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/google/uuid"
)

type AskAccessAssistantTool struct {
	conversationID string // Store conversation ID to maintain context
}

func (t *AskAccessAssistantTool) Name() string {
	return "ask_access_assistant"
}

func (t *AskAccessAssistantTool) Description() string {
	return `Ask the Apono Access Assistant for help requesting access to ANY resource.

⭐ CRITICAL: ALWAYS use this tool when you encounter ANY access or permission error!

WHEN TO USE:
1. When status="needs_access" in list_available_resources
2. When ANY operation fails with access/permission errors
3. When you need different or higher permissions than you currently have
4. When user asks to access any resource (database, cluster, cloud, SaaS tool)

APONO CAN GRANT ACCESS TO:
- Databases (PostgreSQL, MySQL, MongoDB, etc.) - read, write, admin access
- Kubernetes clusters - namespaces, pods, deployments, admin access
- Cloud providers (AWS, GCP, Azure) - services, resources, IAM roles
- SaaS tools (GitHub, Jira, etc.) - various permission levels
- ANY integration shown in list_available_resources

🎯 HOW TO PROVIDE CONTEXT (CRITICAL):
When calling this tool, include ALL technical details you have. Generic requests may give wrong access!

BE SPECIFIC - Include:
✓ Exact database/resource name (e.g., "apono database" not just "postgres")
✓ Specific tables/schemas you need (e.g., "users table in apono database")
✓ What operation you're trying (e.g., "INSERT into sessions table")
✓ Error messages you got (e.g., "permission denied for table access_requests")
✓ Integration ID if you know it (from list_resources_filtered)
✓ Any technical context from previous operations

GOOD Examples:
✓ "I need write access to the 'users' table in the 'apono' database on local-postgres"
✓ "I got 'permission denied for table access_requests' when trying to INSERT in the apono database"
✓ "I need to query the sessions table in database 'apono' on integration local-postgres"

BAD Examples (too vague):
✗ "I need database access" (which database?)
✗ "I need write access" (to what?)
✗ "I got permission denied" (on what resource? what operation?)

WHAT IT DOES:
1. Understands what access you need from the context you provide
2. Asks clarifying questions if needed (keep calling with answers)
3. Builds the correct access request for you
4. If access is NOT available via Apono, the assistant will tell you

💡 WHEN NOT SURE - USE EXPLORATION TOOLS:
If the assistant asks for clarification OR you lack specific details about resources:
1. Use list_resources_filtered to find exact resource names and IDs
   Example: {"integration_type": "postgresql"} or {"integration_name": "apono"}
2. Provide those specific details in your next message to this tool
3. This helps avoid requesting access to the wrong resource

RESPONSE FIELDS:
- text_response: Message from assistant (always show this to user)
- has_request_cta: true when assistant has built a complete request
- entitlements: List of resources/permissions to request (when has_request_cta=true)

NEXT STEPS:
- If has_request_cta=false: Show text_response to user, call again with their answer if asked
- If has_request_cta=true: Extract entitlements and call create_access_request tool
- If create_access_request fails: Read the error, call this tool again with error details to get help fixing it

⚠️ AFTER REQUESTING ACCESS - VERIFY IT WORKED:
1. After access is granted, try your operation again
2. If you STILL get permission errors, you may have requested wrong access!
3. Use get_request_details with the request_id to see what was actually granted
   - Check: "Did I get access to the right database/resource?"
   - Check: "Do I have the right permissions (READ vs WRITE)?"
4. Call this tool again with MORE SPECIFIC details:
   - "I requested access to postgres but still get 'permission denied for table users in apono database'"
   - "I got access but it's read-only, I need write access to INSERT data"
   - Include findings from get_request_details in your message
5. The assistant will help you request the CORRECT access

ERROR RECOVERY:
⚠️ If create_access_request fails with validation errors:
1. ALWAYS call this tool again explaining what went wrong
2. Example: "The access request failed with error: justification required. What should I provide?"
3. The assistant will help you fix the request parameters`
}

func (t *AskAccessAssistantTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"message": map[string]interface{}{
				"type":        "string",
				"description": "Detailed description including ALL technical context: exact database/resource name, specific tables/schemas, operation you're trying, error messages, integration name. BE SPECIFIC! Example: 'I need write access to the users table in the apono database on local-postgres integration' NOT just 'I need database access'",
			},
			"integration_id": map[string]interface{}{
				"type":        "string",
				"description": "Optional: The integration ID to request access to. If provided, helps the assistant be more specific.",
			},
		},
		"required": []string{"message"},
	}
}

type AskAccessAssistantArgs struct {
	Message       string `json:"message"`
	IntegrationID string `json:"integration_id,omitempty"`
}

func (t *AskAccessAssistantTool) Execute(ctx context.Context, client *aponoapi.AponoClient, arguments json.RawMessage) (interface{}, error) {
	var args AskAccessAssistantArgs
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Build message with optional integration context
	userMessage := args.Message
	if args.IntegrationID != "" {
		userMessage = fmt.Sprintf("Integration ID: %s\n%s", args.IntegrationID, args.Message)
	}

	// Initialize conversation ID if not set
	conversationID := t.conversationID
	if conversationID == "" {
		// Create a new conversation with a UUID
		conversationID = uuid.New().String()
		t.conversationID = conversationID
	}

	// Send message to assistant
	request := clientapi.NewAssistantMessageRequestModel(
		"user_message",
		userMessage,
		conversationID,
	)

	// Check if we need to use a different base URL for the assistant API
	assistantClient := client.ClientAPI
	assistantAPIURL := os.Getenv("APONO_ASSISTANT_API_URL")
	fmt.Printf("[DEBUG] APONO_ASSISTANT_API_URL env var value: '%s'\n", assistantAPIURL)
	if assistantAPIURL != "" {
		// Create a new client configuration with the assistant API URL
		fmt.Printf("[DEBUG] Using custom assistant API URL from APONO_ASSISTANT_API_URL: %s\n", assistantAPIURL)

		// Copy the original configuration to preserve authentication and other settings
		originalConfig := client.ClientAPI.GetConfig()
		assistantConfig := clientapi.NewConfiguration()
		assistantConfig.Servers = clientapi.ServerConfigurations{
			{
				URL: assistantAPIURL,
			},
		}

		// Wrap the HTTP client to add userId query parameter
		assistantConfig.HTTPClient = &http.Client{
			Transport: &userIdTransport{
				Transport: originalConfig.HTTPClient.Transport,
				UserID:    client.Session.UserID,
			},
		}

		assistantConfig.UserAgent = originalConfig.UserAgent
		assistantConfig.DefaultHeader = originalConfig.DefaultHeader

		assistantClient = clientapi.NewAPIClient(assistantConfig)
	} else {
		// Also wrap the default client to add userId
		originalConfig := client.ClientAPI.GetConfig()
		wrappedClient := &http.Client{
			Transport: &userIdTransport{
				Transport: originalConfig.HTTPClient.Transport,
				UserID:    client.Session.UserID,
			},
		}

		assistantConfig := clientapi.NewConfiguration()
		assistantConfig.Scheme = originalConfig.Scheme
		assistantConfig.Host = originalConfig.Host
		assistantConfig.HTTPClient = wrappedClient
		assistantConfig.UserAgent = originalConfig.UserAgent
		assistantConfig.DefaultHeader = originalConfig.DefaultHeader

		assistantClient = clientapi.NewAPIClient(assistantConfig)
	}

	// Log the API call
	config := assistantClient.GetConfig()
	var baseURL string
	if len(config.Servers) > 0 && config.Servers[0].URL != "" {
		baseURL = config.Servers[0].URL
	} else {
		baseURL = fmt.Sprintf("%s://%s", config.Scheme, config.Host)
	}
	endpoint := fmt.Sprintf("%s/api/client/v1/assistant/chat?userId=%s", baseURL, client.Session.UserID)
	fmt.Printf("[DEBUG] Calling assistant API: POST %s\n", endpoint)
	fmt.Printf("[DEBUG] User ID: %s\n", client.Session.UserID)
	fmt.Printf("[DEBUG] Conversation ID: %s\n", conversationID)
	fmt.Printf("[DEBUG] Message: %s\n", userMessage)

	response, httpResp, err := assistantClient.AccessAssistantAPI.SendMessageToAssistant(ctx).
		AssistantMessageRequestModel(*request).
		Execute()

	if httpResp != nil {
		fmt.Printf("[DEBUG] Response Status: %d\n", httpResp.StatusCode)
	}

	if err != nil {
		if httpResp != nil && httpResp.StatusCode >= 400 {
			fmt.Printf("[DEBUG] Response Headers: %v\n", httpResp.Header)
			return nil, fmt.Errorf("assistant API error (status %d): %w", httpResp.StatusCode, err)
		}
		return nil, fmt.Errorf("failed to send message to assistant: %w", err)
	}

	// Extract the assistant's response
	assistantMessage := response.GetMessage()
	dataItems := assistantMessage.GetData()

	result := map[string]interface{}{
		"conversation_id": t.conversationID,
		"text_response":   extractTextFromData(dataItems),
		"has_request_cta": false,
	}

	// Look for request CTA in the response
	for _, dataItem := range dataItems {
		if dataItem.HasClientRequestCta() {
			requestCta := dataItem.GetClientRequestCta()

			// Check for resources request
			if requestCta.HasResourcesRequest() {
				resourcesReq := requestCta.GetResourcesRequest()

				result["has_request_cta"] = true
				result["valid_request"] = resourcesReq.GetValidRequest()
				result["requires_approval"] = resourcesReq.GetRequiresApproval()
				result["justification"] = resourcesReq.GetJustification()

				// Extract entitlements
				entitlements := resourcesReq.GetEntitlements()
				result["entitlements"] = formatEntitlements(entitlements)

				// Build the CLI command to create the request
				if resourcesReq.GetValidRequest() {
					result["cli_command"] = buildRequestCommand(entitlements, resourcesReq.GetJustification())
				}
			}

			// Check for bundles request
			if requestCta.HasBundlesRequest() {
				bundlesReq := requestCta.GetBundlesRequest()
				result["has_bundle_request"] = true
				result["bundles"] = bundlesReq
			}
		}
	}

	// Add suggestions if available
	suggestions := response.GetSuggestions()
	if len(suggestions) > 0 {
		suggestionTexts := make([]string, 0, len(suggestions))
		for _, suggestion := range suggestions {
			suggestionTexts = append(suggestionTexts, suggestion.Title)
		}
		result["suggestions"] = suggestionTexts
	}

	return result, nil
}

func extractTextFromData(dataItems []clientapi.AssistantMessageDataClientModel) string {
	var text string
	for _, dataItem := range dataItems {
		if dataItem.HasMarkdown() {
			markdown := dataItem.GetMarkdown()
			text += markdown.Content + "\n"
		}
	}
	return text
}

func formatEntitlements(entitlements []clientapi.AccessUnitClientModel) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(entitlements))

	for _, entitlement := range entitlements {
		resource := entitlement.Resource
		permission := entitlement.Permission

		item := map[string]interface{}{
			"integration_id":     resource.Integration.Id,
			"integration_name":   resource.Integration.Name,
			"resource_type":      resource.Type.Id,
			"resource_type_name": resource.Type.Name,
			"resource_id":        resource.Id,
			"resource_name":      resource.Name,
			"permission_id":      permission.Id,
			"permission_name":    permission.Name,
		}

		result = append(result, item)
	}

	return result
}

func buildRequestCommand(entitlements []clientapi.AccessUnitClientModel, justification string) string {
	if len(entitlements) == 0 {
		return ""
	}

	// Group entitlements by integration and resource type
	type key struct {
		integrationID string
		resourceType  string
	}

	groups := make(map[key]struct {
		permissions []string
		resources   []string
	})

	for _, ent := range entitlements {
		k := key{
			integrationID: ent.Resource.Integration.Id,
			resourceType:  ent.Resource.Type.Id,
		}

		group := groups[k]
		group.permissions = append(group.permissions, ent.Permission.Id)
		group.resources = append(group.resources, ent.Resource.Id)
		groups[k] = group
	}

	// Build command for first group (simplified)
	var cmd strings.Builder
	cmd.WriteString("apono requests new")

	for k, group := range groups {
		cmd.WriteString(fmt.Sprintf(" --integration %s", k.integrationID))
		cmd.WriteString(fmt.Sprintf(" --resource-type %s", k.resourceType))

		if len(group.permissions) > 0 {
			cmd.WriteString(" --permissions")
			for _, perm := range group.permissions {
				cmd.WriteString(fmt.Sprintf(" %s", perm))
			}
		}

		if len(group.resources) > 0 {
			cmd.WriteString(" --resources")
			for _, res := range group.resources {
				cmd.WriteString(fmt.Sprintf(" %s", res))
			}
		}

		// Only use first group for simplicity
		break
	}

	if justification != "" {
		cmd.WriteString(fmt.Sprintf(" --justification \"%s\"", justification))
	}

	return cmd.String()
}

// userIdTransport is a custom HTTP transport that adds userId query parameter to requests
type userIdTransport struct {
	Transport http.RoundTripper
	UserID    string
}

func (t *userIdTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Add userId query parameter
	q := req.URL.Query()
	q.Add("userId", t.UserID)
	req.URL.RawQuery = q.Encode()

	// Use the underlying transport or default if nil
	transport := t.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	return transport.RoundTrip(req)
}
