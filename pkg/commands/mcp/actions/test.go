package actions

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/commands/mcp/tools"
)

func MCPTest() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test MCP tools directly",
		Long:  "Test MCP tools directly from the CLI to debug their behavior",
	}

	cmd.AddCommand(testListAvailableResources())
	cmd.AddCommand(testSetupDatabaseMCP())
	cmd.AddCommand(testSetupDatabaseMCPV2())
	cmd.AddCommand(testAskAccessAssistant())
	cmd.AddCommand(testCreateAccessRequest())
	cmd.AddCommand(testGetRequestDetails())
	cmd.AddCommand(testListResourcesFiltered())
	cmd.AddCommand(testShowConfig())

	return cmd
}

func testListAvailableResources() *cobra.Command {
	return &cobra.Command{
		Use:   "list-available-resources",
		Short: "Test the list_available_resources tool",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.CreateClient(cmd.Context(), "")
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			tool := &tools.ListAvailableResourcesTool{}
			result, err := tool.Execute(context.Background(), client, nil)
			if err != nil {
				return fmt.Errorf("tool execution failed: %w", err)
			}

			output, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal result: %w", err)
			}

			fmt.Println(string(output))
			return nil
		},
	}
}

func testSetupDatabaseMCP() *cobra.Command {
	var sessionID string

	cmd := &cobra.Command{
		Use:   "setup-database-mcp",
		Short: "Test the setup_database_mcp tool",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sessionID == "" {
				return fmt.Errorf("--session-id is required")
			}

			client, err := aponoapi.CreateClient(cmd.Context(), "")
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			tool := &tools.SetupDatabaseMCPTool{}
			arguments, _ := json.Marshal(map[string]string{
				"session_id": sessionID,
			})

			result, err := tool.Execute(context.Background(), client, arguments)
			if err != nil {
				return fmt.Errorf("tool execution failed: %w", err)
			}

			output, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal result: %w", err)
			}

			fmt.Println(string(output))
			return nil
		},
	}

	cmd.Flags().StringVar(&sessionID, "session-id", "", "Session ID to setup MCP for")
	cmd.MarkFlagRequired("session-id")

	return cmd
}

func testSetupDatabaseMCPV2() *cobra.Command {
	var sessionID string

	cmd := &cobra.Command{
		Use:   "setup-database-mcp-v2",
		Short: "Test the setup_database_mcp_v2 tool",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sessionID == "" {
				return fmt.Errorf("--session-id is required")
			}

			client, err := aponoapi.CreateClient(cmd.Context(), "")
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			tool := &tools.SetupDatabaseMCPV2Tool{}
			arguments, _ := json.Marshal(map[string]string{
				"session_id": sessionID,
			})

			result, err := tool.Execute(context.Background(), client, arguments)
			if err != nil {
				return fmt.Errorf("tool execution failed: %w", err)
			}

			output, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal result: %w", err)
			}

			fmt.Println(string(output))
			return nil
		},
	}

	cmd.Flags().StringVar(&sessionID, "session-id", "", "Session ID to setup MCP for")
	cmd.MarkFlagRequired("session-id")

	return cmd
}

func testAskAccessAssistant() *cobra.Command {
	var message string
	var integrationID string

	cmd := &cobra.Command{
		Use:   "ask-access-assistant",
		Short: "Test the ask_access_assistant tool",
		RunE: func(cmd *cobra.Command, args []string) error {
			if message == "" {
				return fmt.Errorf("--message is required")
			}

			client, err := aponoapi.CreateClient(cmd.Context(), "")
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			tool := &tools.AskAccessAssistantTool{}
			arguments, _ := json.Marshal(map[string]string{
				"message":        message,
				"integration_id": integrationID,
			})

			result, err := tool.Execute(context.Background(), client, arguments)
			if err != nil {
				return fmt.Errorf("tool execution failed: %w", err)
			}

			output, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal result: %w", err)
			}

			fmt.Println(string(output))
			return nil
		},
	}

	cmd.Flags().StringVar(&message, "message", "", "Message to send to the assistant")
	cmd.Flags().StringVar(&integrationID, "integration-id", "", "Optional integration ID")
	cmd.MarkFlagRequired("message")

	return cmd
}

func testCreateAccessRequest() *cobra.Command {
	var bundleID string
	var integrationID string
	var resourceTypeID string
	var resourceIDs []string
	var permissionIDs []string
	var justification string
	var durationHours float64

	cmd := &cobra.Command{
		Use:   "create-access-request",
		Short: "Test the create_access_request tool",
		RunE: func(cmd *cobra.Command, args []string) error {
			if bundleID == "" && integrationID == "" {
				return fmt.Errorf("either --bundle-id or --integration-id must be provided")
			}

			client, err := aponoapi.CreateClient(cmd.Context(), "")
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			tool := &tools.CreateAccessRequestTool{}

			requestArgs := map[string]interface{}{}
			if bundleID != "" {
				requestArgs["bundle_id"] = bundleID
			}
			if integrationID != "" {
				requestArgs["integration_id"] = integrationID
			}
			if resourceTypeID != "" {
				requestArgs["resource_type_id"] = resourceTypeID
			}
			if len(resourceIDs) > 0 {
				requestArgs["resource_ids"] = resourceIDs
			}
			if len(permissionIDs) > 0 {
				requestArgs["permission_ids"] = permissionIDs
			}
			if justification != "" {
				requestArgs["justification"] = justification
			}
			if durationHours > 0 {
				requestArgs["duration_hours"] = durationHours
			}

			arguments, _ := json.Marshal(requestArgs)

			result, err := tool.Execute(context.Background(), client, arguments)
			if err != nil {
				return fmt.Errorf("tool execution failed: %w", err)
			}

			output, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal result: %w", err)
			}

			fmt.Println(string(output))
			return nil
		},
	}

	cmd.Flags().StringVar(&bundleID, "bundle-id", "", "Bundle ID for bundle-based request")
	cmd.Flags().StringVar(&integrationID, "integration-id", "", "Integration ID for integration-based request")
	cmd.Flags().StringVar(&resourceTypeID, "resource-type-id", "", "Resource type ID")
	cmd.Flags().StringSliceVar(&resourceIDs, "resource-ids", []string{}, "Resource IDs (comma-separated)")
	cmd.Flags().StringSliceVar(&permissionIDs, "permission-ids", []string{}, "Permission IDs (comma-separated)")
	cmd.Flags().StringVar(&justification, "justification", "", "Justification for the request")
	cmd.Flags().Float64Var(&durationHours, "duration-hours", 0, "Duration in hours")

	return cmd
}

func testGetRequestDetails() *cobra.Command {
	var requestID string

	cmd := &cobra.Command{
		Use:   "get-request-details",
		Short: "Test the get_request_details tool",
		RunE: func(cmd *cobra.Command, args []string) error {
			if requestID == "" {
				return fmt.Errorf("--request-id is required")
			}

			client, err := aponoapi.CreateClient(cmd.Context(), "")
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			tool := &tools.GetRequestDetailsTool{}
			arguments, _ := json.Marshal(map[string]string{
				"request_id": requestID,
			})

			result, err := tool.Execute(context.Background(), client, arguments)
			if err != nil {
				return fmt.Errorf("tool execution failed: %w", err)
			}

			output, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal result: %w", err)
			}

			fmt.Println(string(output))
			return nil
		},
	}

	cmd.Flags().StringVar(&requestID, "request-id", "", "Access request ID (e.g., AR-00001)")
	cmd.MarkFlagRequired("request-id")

	return cmd
}

func testListResourcesFiltered() *cobra.Command {
	var integrationType string
	var integrationName string

	cmd := &cobra.Command{
		Use:   "list-resources-filtered",
		Short: "Test the list_resources_filtered tool",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.CreateClient(cmd.Context(), "")
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			tool := &tools.ListResourcesFilteredTool{}

			filterArgs := map[string]interface{}{}
			if integrationType != "" {
				filterArgs["integration_type"] = integrationType
			}
			if integrationName != "" {
				filterArgs["integration_name"] = integrationName
			}

			arguments, _ := json.Marshal(filterArgs)

			result, err := tool.Execute(context.Background(), client, arguments)
			if err != nil {
				return fmt.Errorf("tool execution failed: %w", err)
			}

			output, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal result: %w", err)
			}

			fmt.Println(string(output))
			return nil
		},
	}

	cmd.Flags().StringVar(&integrationType, "integration-type", "", "Filter by integration type (e.g., postgresql, kubernetes)")
	cmd.Flags().StringVar(&integrationName, "integration-name", "", "Filter by integration name (e.g., local-postgres)")

	return cmd
}

func testShowConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-config",
		Short: "Show current API configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.CreateClient(cmd.Context(), "")
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			config := client.ClientAPI.GetConfig()

			var baseURL string
			if len(config.Servers) > 0 && config.Servers[0].URL != "" {
				baseURL = config.Servers[0].URL
			} else {
				baseURL = fmt.Sprintf("%s://%s", config.Scheme, config.Host)
			}

			fmt.Println("=== API Configuration ===")
			fmt.Printf("Full Base URL: %s\n", baseURL)
			fmt.Printf("Scheme: %s\n", config.Scheme)
			fmt.Printf("Host: %s\n", config.Host)
			fmt.Println("\n=== Endpoints ===")
			fmt.Printf("Assistant API: POST %s/api/client/v1/assistant/chat\n", baseURL)
			fmt.Printf("Dry-run API: POST %s/api/client/v1/access-requests/dry-run\n", baseURL)
			fmt.Printf("Create Request API: POST %s/api/client/v1/access-requests\n", baseURL)
			fmt.Printf("List Integrations API: GET %s/api/client/v1/integrations\n", baseURL)

			return nil
		},
	}

	return cmd
}
