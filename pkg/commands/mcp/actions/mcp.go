package actions

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/commands/mcp/approval"
	"github.com/apono-io/apono-cli/pkg/commands/mcp/proxy"
	"github.com/apono-io/apono-cli/pkg/commands/mcp/registry"
	"github.com/apono-io/apono-cli/pkg/commands/mcp/risk"
	"github.com/apono-io/apono-cli/pkg/commands/mcp/targets"
	"github.com/apono-io/apono-cli/pkg/groups"
	"github.com/apono-io/apono-cli/pkg/utils"
)

const (
	McpEndpointPath      = "/api/client/v1/mcp"
	EmptyErrorStatusCode = 0
	debugFlagName        = "debug"
	mcpMethodInitialize  = "initialize"

	ErrorCodeAuthenticationFailed = -32001
	ErrorCodeAuthorizationFailed  = -32003
	ErrorCodeInternalError        = -32603
)

type InitializeClientParams struct {
	Params struct {
		ClientInfo struct {
			Name string `json:"name"`
		} `json:"clientInfo"`
	} `json:"params"`
}

const (
	proxyFlagName            = "proxy"
	targetsFileFlagName      = "targets-file"
	allIntegrationsFlagName  = "all-integrations"
	riskActionFlagName       = "risk-action"
	mcpServersFileFlagName   = "mcp-servers-file"
)

func MCP() *cobra.Command {
	var debug bool
	var proxyEnabled bool
	var targetsFilePath string
	var allIntegrations bool
	var riskAction string
	var mcpServersFile string

	cmd := &cobra.Command{
		Use:     "mcp",
		Short:   "MCP server and tools for managing database access",
		GroupID: groups.OtherCommandsGroup.ID,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := utils.InitMcpLogFile(); err != nil {
				return fmt.Errorf("failed to initialize MCP log file: %w", err)
			}
			defer utils.CloseMcpLogFile()

			utils.McpLogf("=== Apono MCP STDIO Server Starting ===")

			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to get API client from context: %w", err)
			}

			if proxyEnabled {
				utils.McpLogf("Proxy mode enabled, risk-action=%s", riskAction)
				return runLocalSTDIOServerWithProxy(client, debug, targetsFilePath, allIntegrations, riskAction, mcpServersFile)
			}

			utils.McpLogf("Ready to receive requests...")
			return runLocalSTDIOServer(client, debug)
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&debug, debugFlagName, false, "Enable debug logging for request/response bodies")
	flags.BoolVar(&proxyEnabled, proxyFlagName, false, "Enable local MCP proxy mode with dynamic backend spawning")
	flags.StringVar(&targetsFilePath, targetsFileFlagName, "", "Path to targets.yaml file (default: ~/.apono/mcp-proxy/targets.yaml)")
	flags.BoolVar(&allIntegrations, allIntegrationsFlagName, false, "Show all integrations as targets, not just database types")
	flags.StringVar(&riskAction, riskActionFlagName, "deny", "Action for risky operations: 'deny' (block), 'approve' (request approval via Apono), 'allow' (skip risk checks)")
	flags.StringVar(&mcpServersFile, mcpServersFileFlagName, "", "Path to mcp-servers.yaml (default: ~/.apono/mcp-servers.yaml)")

	// Add test subcommand
	cmd.AddCommand(MCPTest())

	return cmd
}

func runLocalSTDIOServer(client *aponoapi.AponoClient, debug bool) error {
	scanner := bufio.NewScanner(os.Stdin)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	utils.McpLogf("=== STDIO Server Started, waiting for input ===")
	defer func() {
		utils.McpLogf("=== STDIO Server shutting down ===")
	}()

	handler := NewMCPHandler(client)

	lineCh := make(chan string)
	errsCh := make(chan error, 1)

	go func() {
		defer close(lineCh)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			lineCh <- line
		}
		if err := scanner.Err(); err != nil {
			errsCh <- fmt.Errorf("error reading stdin: %w", err)
		}
		close(errsCh)
	}()

	for {
		select {
		case <-ctx.Done():
			return nil

		case err := <-errsCh:
			if err != nil {
				utils.McpLogf("[Error]: Scanner error: %v", err)
				return err
			}
			return nil

		case line, ok := <-lineCh:
			if !ok {
				return nil
			}
			if line == "" {
				continue
			}

			if debug {
				utils.McpLogf("[Debug]: Request: %s", line)
			}

			response := handler.HandleRequest(ctx, line)

			if debug {
				utils.McpLogf("[Debug]: Response: %s", response)
			}

			// Only send response if there is one (empty string means notification with no response)
			if response != "" {
				fmt.Println(response)
			}
		}
	}
}

func runLocalSTDIOServerWithProxy(client *aponoapi.AponoClient, debug bool, targetsFilePath string, allIntegrations bool, riskAction string, mcpServersFile string) error {
	scanner := bufio.NewScanner(os.Stdin)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Resolve targets file path
	if targetsFilePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		targetsFilePath = filepath.Join(homeDir, ".apono", "mcp-proxy", "targets.yaml")
	}

	utils.McpLogf("Targets file: %s", targetsFilePath)

	// Resolve MCP servers config path and load registry
	if mcpServersFile == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		mcpServersFile = filepath.Join(homeDir, ".apono", "mcp-servers.yaml")
	}

	var mcpReg *registry.MCPServersConfig
	loadedReg, err := registry.LoadMCPServersConfig(mcpServersFile)
	if err != nil {
		utils.McpLogf("Could not load MCP servers config from %s: %v, using built-in defaults", mcpServersFile, err)
		mcpReg = registry.DefaultConfig()
	} else {
		utils.McpLogf("Loaded MCP servers config from %s (%d servers)", mcpServersFile, len(loadedReg.Servers))
		mcpReg = loadedReg
	}

	// Build target sources
	fileLoader := targets.NewFileTargetLoader(targetsFilePath)
	sessionProvider := targets.NewSessionTargetProvider(client, allIntegrations)
	// File targets take priority (first source wins on conflict)
	compositeSource := targets.NewCompositeTargetSource(fileLoader, sessionProvider)

	// Build API base URL and HTTP client
	apiCfg := client.ClientAPI.GetConfig()
	apiBaseURL := fmt.Sprintf("%s://%s", apiCfg.Scheme, apiCfg.Host)

	// Configure risk detection and approval based on --risk-action flag
	var riskDetector risk.RiskDetector
	var approver approval.Approver

	switch riskAction {
	case "allow":
		// No risk detection — all operations are allowed
		utils.McpLogf("Risk action: allow (no risk checks)")
	case "approve":
		// Risk detection + Apono action-approval API flow
		riskDetector = risk.NewPatternRiskDetector(risk.DefaultRiskConfig())
		baseApprover := approval.NewAponoActionApprover(apiBaseURL, apiCfg.HTTPClient, client.Session.UserID, 5*time.Minute)
		approver = approval.NewApprovalCache(baseApprover)
		utils.McpLogf("Risk action: approve (risky ops require Apono approval, with intent/pattern caching)")
	default: // "deny"
		// Risk detection, block without approval
		riskDetector = risk.NewPatternRiskDetector(risk.DefaultRiskConfig())
		utils.McpLogf("Risk action: deny (risky ops blocked)")
	}

	// Create proxy manager
	pm := proxy.NewLocalProxyManager(proxy.LocalProxyManagerConfig{
		MCPRegistry:     mcpReg,
		TargetSource:    compositeSource,
		RiskDetector:    riskDetector,
		Approver:        approver,
		APIBaseURL:      apiBaseURL,
		HTTPClient:      apiCfg.HTTPClient,
		TargetsFilePath: targetsFilePath,
	})
	defer pm.Close()

	// Start session watcher in background
	watcherCtx, watcherCancel := context.WithCancel(ctx)
	defer watcherCancel()
	go pm.SessionWatcher().Start(watcherCtx)

	// Start background cleanup
	pm.StartCleanupRoutine()

	// Wire tools/list_changed notification
	pm.SetToolsChangedCallback(func() {
		notification := `{"jsonrpc":"2.0","method":"notifications/tools/list_changed"}`
		fmt.Println(notification)
		utils.McpLogf("Sent notifications/tools/list_changed")
	})

	utils.McpLogf("=== STDIO Server with Proxy Started, waiting for input ===")
	defer func() {
		utils.McpLogf("=== STDIO Server with Proxy shutting down ===")
	}()

	handler := NewMCPHandlerWithProxy(client, pm)

	lineCh := make(chan string)
	errsCh := make(chan error, 1)

	go func() {
		defer close(lineCh)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			lineCh <- line
		}
		if err := scanner.Err(); err != nil {
			errsCh <- fmt.Errorf("error reading stdin: %w", err)
		}
		close(errsCh)
	}()

	for {
		select {
		case <-ctx.Done():
			return nil

		case err := <-errsCh:
			if err != nil {
				utils.McpLogf("[Error]: Scanner error: %v", err)
				return err
			}
			return nil

		case line, ok := <-lineCh:
			if !ok {
				return nil
			}
			if line == "" {
				continue
			}

			// Always log the method of incoming requests
			var peek struct {
				Method string `json:"method"`
			}
			if json.Unmarshal([]byte(line), &peek) == nil && peek.Method != "" {
				utils.McpLogf("[STDIO] >> %s", peek.Method)
			}

			if debug {
				utils.McpLogf("[Debug]: Request: %s", line)
			}

			response := handler.HandleRequest(ctx, line)

			if debug {
				utils.McpLogf("[Debug]: Response: %s", response)
			}

			if response != "" {
				utils.McpLogf("[STDIO] << %s (response len=%d)", peek.Method, len(response))
				fmt.Println(response)
			}
		}
	}
}

func runSTDIOServer(endpoint string, httpClient *http.Client, debug bool) error {
	scanner := bufio.NewScanner(os.Stdin)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	utils.McpLogf("=== STDIO Server Started, waiting for input ===")
	defer func() {
		utils.McpLogf("=== STDIO Server shutting down ===")
	}()

	var clientName string

	lineCh := make(chan string)
	errsCh := make(chan error, 1)

	go func() {
		defer close(lineCh)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			lineCh <- line
		}
		if err := scanner.Err(); err != nil {
			errsCh <- fmt.Errorf("error reading stdin: %w", err)
		}
		close(errsCh)
	}()

	for {
		select {
		case <-ctx.Done():
			return nil

		case err := <-errsCh:
			if err != nil {
				utils.McpLogf("[Error]: Scanner error: %v", err)
				return err
			}
			return nil

		case line, ok := <-lineCh:
			if !ok {
				return nil
			}
			if line == "" {
				continue
			}

			var requestData map[string]interface{}
			if err := json.Unmarshal([]byte(line), &requestData); err == nil {
				if method, ok := requestData["method"].(string); ok {
					utils.McpLogf("Received request method: %q", method)
					if debug {
						utils.McpLogf("[Debug]: Request body: %s", line)
					}
					if strings.ToLower(method) == mcpMethodInitialize {
						var name string
						name, err = extractClientNameFromInitializeRequest(requestData)
						if err != nil {
							utils.McpLogf("[Error]: Failed to extract client name: %v", err)
						} else if name != "" {
							clientName = name
							utils.McpLogf("Client name set to: %s", clientName)
						}
					}
				}
			}

			response, statusCode := sendMcpRequest(endpoint, httpClient, line, clientName, debug)

			if statusCode == EmptyErrorStatusCode {
				utils.McpLogf("[Error]: Failed to process request, sending error response")
				errorResponse := createErrorResponse(ErrorCodeInternalError, "Internal error", "Failed to process request")
				fmt.Println(errorResponse)
				continue
			}

			propagateResponseToStdout(response, statusCode)
		}
	}
}

func extractClientNameFromInitializeRequest(requestData map[string]interface{}) (string, error) {
	data, err := json.Marshal(requestData)
	if err != nil {
		return "", err
	}
	var params InitializeClientParams
	if err = json.Unmarshal(data, &params); err != nil {
		return "", err
	}
	return params.Params.ClientInfo.Name, nil
}

func sendMcpRequest(endpoint string, httpClient *http.Client, request string, userAgent string, debug bool) (string, int) {
	if debug {
		utils.McpLogf("[Debug]: Sending request to endpoint: %s", endpoint)
	}

	httpReq, err := http.NewRequestWithContext(context.Background(), "POST", endpoint, bytes.NewBuffer([]byte(request)))
	if err != nil {
		utils.McpLogf("[Error]: Failed to create HTTP request: %v", err)
		return "", EmptyErrorStatusCode
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if userAgent != "" {
		httpReq.Header.Set("User-Agent", userAgent)
	}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		utils.McpLogf("[Error]: Failed to send HTTP request: %v", err)
		return "", EmptyErrorStatusCode
	}
	defer func(Body io.ReadCloser) {
		closeErr := Body.Close()
		if closeErr != nil {
			utils.McpLogf("WARNING: Failed to close response body: %v", closeErr)
		}
	}(resp.Body)

	utils.McpLogf("HTTP response status: %d", resp.StatusCode)

	if resp.StatusCode == http.StatusUnauthorized {
		utils.McpLogf("[Error]: Authentication failed - Status: %d", resp.StatusCode)
		errorResponse := createErrorResponse(ErrorCodeAuthenticationFailed, "Authentication failed", "Please run 'apono login' command to authenticate")
		return errorResponse, resp.StatusCode
	}

	if resp.StatusCode == http.StatusForbidden {
		utils.McpLogf("[Error]: Authorization failed - Status: %d", resp.StatusCode)
		errorResponse := createErrorResponse(ErrorCodeAuthorizationFailed, "Authorization failed", "Access forbidden")
		return errorResponse, resp.StatusCode
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.McpLogf("[Error]: Failed to read response body: %v", err)
		return "", resp.StatusCode
	}

	if debug {
		utils.McpLogf("[Debug]: Response body: %s", string(responseBody))
	}

	response := string(responseBody)

	return response, resp.StatusCode
}

func propagateResponseToStdout(response string, statusCode int) {
	if response != "" {
		fmt.Println(response)
		return
	}

	utils.McpLogf("No response body to send, status: %d", statusCode)
}

func createErrorResponse(errorCode int, message, data string) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":null,"error":{"code":%d,"message":"%s","data":"%s"}}`, errorCode, message, data)
}
