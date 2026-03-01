package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/commands/mcp/approval"
	"github.com/apono-io/apono-cli/pkg/commands/mcp/proxy"
	"github.com/apono-io/apono-cli/pkg/commands/mcp/registry"
	"github.com/apono-io/apono-cli/pkg/commands/mcp/targets"
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
	cmd.AddCommand(testApproveFlow())
	cmd.AddCommand(testAutoSpawn())

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

func testApproveFlow() *cobra.Command {
	var targetID string
	var toolName string
	var reason string
	var timeoutMinutes int

	cmd := &cobra.Command{
		Use:   "approve-flow",
		Short: "Test the approval flow end-to-end",
		Long: `Tests the Apono approval flow by:
1. Discovering targets from Apono sessions
2. Finding the integration ID for the specified target
3. Creating an approval request via the Apono API
4. Polling until approved or denied (or timeout)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.CreateClient(cmd.Context(), "")
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			ctx := context.Background()

			// Discover targets
			fmt.Println("=== Discovering targets ===")
			sessionProvider := targets.NewSessionTargetProvider(client, true)
			targetList, err := sessionProvider.ListTargets(ctx)
			if err != nil {
				return fmt.Errorf("failed to list targets: %w", err)
			}

			if len(targetList) == 0 {
				return fmt.Errorf("no targets found — make sure you have integrations in Apono")
			}

			fmt.Printf("Found %d targets:\n", len(targetList))
			for _, t := range targetList {
				fmt.Printf("  - %s (name=%s, type=%s, status=%s)\n", t.ID, t.Name, t.Type, t.Status)
			}

			// Pick target
			if targetID == "" {
				// Pick first available target
				targetID = targetList[0].ID
				fmt.Printf("\nNo --target-id specified, using first target: %s\n", targetID)
			}

			// Get target definition to find integration ID
			fmt.Printf("\n=== Getting target details for %s ===\n", targetID)
			targetDef, err := sessionProvider.GetTarget(ctx, targetID)
			if err != nil {
				// If GetTarget fails (no active session), try to find integration ID from list
				fmt.Printf("Note: GetTarget failed (%v), looking up integration ID directly...\n", err)

				// We need the integration ID — get it by matching target ID to integration name
				targetDef = &targets.TargetDefinition{
					ID:   targetID,
					Name: targetID,
				}

				// Try to find integration ID via EnsureAccess first to create a session
				fmt.Println("Attempting to ensure access...")
				if accessErr := sessionProvider.EnsureAccess(ctx, targetID); accessErr != nil {
					return fmt.Errorf("failed to ensure access for target %s: %w", targetID, accessErr)
				}

				// Retry GetTarget after ensuring access
				targetDef, err = sessionProvider.GetTarget(ctx, targetID)
				if err != nil {
					return fmt.Errorf("failed to get target definition after ensuring access: %w", err)
				}
			}

			integrationID := targetDef.IntegrationID
			if integrationID == "" {
				return fmt.Errorf("no integration ID found for target %s — cannot create approval request", targetID)
			}

			fmt.Printf("Target: %s\n", targetDef.Name)
			fmt.Printf("Type: %s\n", targetDef.Type)
			fmt.Printf("Integration ID: %s\n", integrationID)

			// Create approval request
			timeout := time.Duration(timeoutMinutes) * time.Minute
			if toolName == "" {
				toolName = "query"
			}
			if reason == "" {
				reason = "Test approval flow: simulated risky operation (DROP TABLE)"
			}

			fmt.Printf("\n=== Submitting approval request ===\n")
			fmt.Printf("Tool: %s\n", toolName)
			fmt.Printf("Reason: %s\n", reason)
			fmt.Printf("Timeout: %v\n", timeout)

			cfg := client.ClientAPI.GetConfig()
			baseURL := fmt.Sprintf("%s://%s", cfg.Scheme, cfg.Host)
			approver := approval.NewAponoActionApprover(baseURL, cfg.HTTPClient, client.Session.UserID, timeout)
			req := approval.ApprovalRequest{
				ToolName:      toolName,
				Arguments:     map[string]interface{}{"sql": "DROP TABLE users"},
				Reason:        reason,
				RiskLevel:     "high",
				TargetID:      targetID,
				IntegrationID: integrationID,
			}

			fmt.Println("\nWaiting for approval (check Slack/Apono UI to approve or deny)...")
			approved, err := approver.RequestApproval(ctx, req)
			if err != nil {
				return fmt.Errorf("approval request failed: %w", err)
			}

			fmt.Println()
			if approved {
				fmt.Println("=== RESULT: APPROVED ===")
				fmt.Println("The risky operation was approved.")
			} else {
				fmt.Println("=== RESULT: DENIED ===")
				fmt.Println("The risky operation was denied.")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&targetID, "target-id", "", "Target ID to test with (default: first discovered target)")
	cmd.Flags().StringVar(&toolName, "tool-name", "query", "Simulated tool name for the approval request")
	cmd.Flags().StringVar(&reason, "reason", "", "Reason for the approval request (default: test message)")
	cmd.Flags().IntVar(&timeoutMinutes, "timeout", 5, "Timeout in minutes for waiting for approval")

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

func testAutoSpawn() *cobra.Command {
	var pollSeconds int

	cmd := &cobra.Command{
		Use:   "auto-spawn",
		Short: "Test the auto-spawn flow: list integrations, detect sessions, auto-spawn MCP backends",
		Long: `Tests the full auto-spawn pipeline:
1. Lists available integrations via Apono API
2. Creates a session watcher that monitors for ready sessions
3. When a session is detected, auto-spawns the matching MCP backend
4. Lists the tools provided by the spawned backend

This is useful for verifying the end-to-end flow without running the full MCP proxy.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.CreateClient(cmd.Context(), "")
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
			defer cancel()

			// Step 1: List available integrations
			fmt.Println("=== Step 1: Listing available integrations ===")
			listTool := &tools.ListAvailableResourcesTool{}
			resources, err := listTool.Execute(ctx, client, nil)
			if err != nil {
				return fmt.Errorf("failed to list resources: %w", err)
			}

			output, _ := json.MarshalIndent(resources, "", "  ")
			fmt.Println(string(output))

			// Step 2: Set up session target provider
			fmt.Println("\n=== Step 2: Setting up session watcher ===")
			sessionProvider := targets.NewSessionTargetProvider(client, false)
			mcpReg := registry.DefaultConfig()

			fmt.Printf("Loaded %d MCP server definitions\n", len(mcpReg.Servers))
			for _, s := range mcpReg.Servers {
				fmt.Printf("  - %s (%s): matches %v\n", s.Name, s.ID, s.IntegrationTypes)
			}

			// Step 3: List current targets
			fmt.Println("\n=== Step 3: Current targets ===")
			targetList, err := sessionProvider.ListTargets(ctx)
			if err != nil {
				return fmt.Errorf("failed to list targets: %w", err)
			}

			if len(targetList) == 0 {
				fmt.Println("No targets found. Request access to a database integration first.")
				return nil
			}

			for _, t := range targetList {
				fmt.Printf("  - %s (type: %s, status: %s)\n", t.Name, t.Type, t.Status)
			}

			// Step 4: Start session watcher and wait for auto-spawn
			fmt.Println("\n=== Step 4: Starting session watcher ===")
			fmt.Printf("Polling every %d seconds for up to 5 minutes...\n", pollSeconds)

			spawnedCh := make(chan string, 1)

			watcher := proxy.NewSessionWatcher(proxy.SessionWatcherConfig{
				TargetSource: sessionProvider,
				MCPRegistry:  mcpReg,
				PollInterval: time.Duration(pollSeconds) * time.Second,
				OnNewSession: func(targetID string, serverDef registry.MCPServerDefinition, target *targets.TargetDefinition) {
					fmt.Printf("\n>>> New session detected: %s (type: %s)\n", targetID, serverDef.ID)
					fmt.Printf("    Server: %s %v\n", serverDef.Command, serverDef.Args)

					// Build credentials
					creds, err := registry.BuildCredentials(serverDef, target.Credentials)
					if err != nil {
						fmt.Printf("    Error building credentials: %v\n", err)
						return
					}

					fmt.Printf("    Credentials built: %d keys\n", len(creds))
					for k := range creds {
						fmt.Printf("      - %s: %s...\n", k, truncate(creds[k], 40))
					}

					spawnedCh <- targetID
				},
				OnExpiredSession: func(targetID string) {
					fmt.Printf("\n>>> Session expired: %s\n", targetID)
				},
			})

			go watcher.Start(ctx)

			select {
			case targetID := <-spawnedCh:
				fmt.Printf("\n=== Success: Auto-spawn triggered for %s ===\n", targetID)
				fmt.Println("The session watcher correctly detected the session and would spawn the MCP backend.")
			case <-ctx.Done():
				fmt.Println("\n=== Timeout: No new sessions detected in 5 minutes ===")
				fmt.Println("Make sure you have an active access session for a database integration.")
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&pollSeconds, "poll-interval", 5, "Poll interval in seconds")

	return cmd
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
