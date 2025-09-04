package actions

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/apono-io/apono-cli/pkg/clientapi"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/groups"
	"github.com/apono-io/apono-cli/pkg/utils"
)

const (
	EmptyErrorStatusCode = 0
	debugFlagName        = "debug"

	ErrorCodeAuthenticationFailed = -32001
	ErrorCodeAuthorizationFailed  = -32003
	ErrorCodeInternalError        = -32603
)

func MCP() *cobra.Command {
	var debug bool

	cmd := &cobra.Command{
		Use:     "mcp",
		Short:   "Run stdio MCP server",
		GroupID: groups.OtherCommandsGroup.ID,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := utils.InitMcpLogFile(); err != nil {
				utils.McpLogf("[Error]: Failed to initialize MCP log file: %v", err)
				return fmt.Errorf("failed to initialize MCP log file: %w", err)
			}
			defer utils.CloseMcpLogFile()

			utils.McpLogf("=== Apono MCP STDIO Server Starting ===")

			mcpClient, err := createAponoMCPClient(cmd, debug)
			if err != nil {
				utils.McpLogf("[Error]: Failed to setup MCP server: %v", err)
				return fmt.Errorf("failed to setup MCP server: %w", err)
			}

			utils.McpLogf("Ready to receive requests...")

			return runSTDIOServer(mcpClient, debug)
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&debug, debugFlagName, false, "Enable debug logging for request/response bodies")

	return cmd
}

func createAponoMCPClient(cmd *cobra.Command, debug bool) (clientapi.ApiHandleMcpMethodRequest, error) {
	utils.McpLogf("=== Starting Setup ===")

	client, err := aponoapi.GetClient(cmd.Context())
	if err != nil {
		utils.McpLogf("[Error]: Failed to create Apono client: %v", err)
		return clientapi.ApiHandleMcpMethodRequest{}, fmt.Errorf("failed to create Apono client: %w", err)
	}

	if debug {
		cfg := client.ClientAPI.GetConfig()
		utils.McpLogf("[Debug]: Sending request to host: %s", cfg.Host)
	}
	mcpClient := client.ClientAPI.MCPServerAPI.HandleMcpMethod(cmd.Context())
	utils.McpLogf("=== Setup Finished ===")
	return mcpClient, nil
}

func runSTDIOServer(mcpClient clientapi.ApiHandleMcpMethodRequest, debug bool) error {
	scanner := bufio.NewScanner(os.Stdin)
	var clientUserAgent string = "apono-cli-mcp-server" // default fallback

	utils.McpLogf("=== STDIO Server Started, waiting for input ===")

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		mcpRequest, method, err := parseMcpRequest(line)
		if err != nil {
			utils.McpLogf("[Error]: %v", err)
			errorResponse := createMcpErrorResponse(ErrorCodeInternalError, "Internal error", "Failed to parse request")
			propagateResponseToStdout(errorResponse)
			continue
		}

		utils.McpLogf("Received request method: \"%s\"", method)
		if debug {
			utils.McpLogf("[Debug]: Request body: %s", line)
		}

		// Update user agent from initialize request
		if method == "initialize" {
			if userAgent := getClientName(mcpRequest); userAgent != "" {
				clientUserAgent = userAgent
			}
		}

		response, httpResponse, err := mcpClient.UserAgent(clientUserAgent).McpRequest(mcpRequest).Execute()
		if err != nil {
			utils.McpLogf("[Error]: MCP client request failed: %v", err)
			errorResponse := createMcpErrorResponse(ErrorCodeInternalError, "Internal error", "Failed to process request")
			propagateResponseToStdout(errorResponse)
			continue
		}

		if debug {
			utils.McpLogf("[Debug]: HTTP response status: %d", httpResponse.StatusCode)
			if response != nil {
				responseBytes, _ := json.Marshal(response)
				utils.McpLogf("[Debug]: Response body: %s", string(responseBytes))
			}
		}

		if errorResponse := handleMcpResponseState(httpResponse); errorResponse != nil {
			propagateResponseToStdout(errorResponse)
			continue
		}

		propagateResponseToStdout(response)
	}

	if err := scanner.Err(); err != nil {
		utils.McpLogf("[Error]: Scanner error: %v", err)
		return fmt.Errorf("error reading stdin: %w", err)
	}

	return nil
}

func propagateResponseToStdout(response *clientapi.McpResponse) {
	fmt.Println(response)
}

func handleMcpResponseState(httpResponse *http.Response) *clientapi.McpResponse {
	statusCode := httpResponse.StatusCode

	switch statusCode {
	case http.StatusUnauthorized:
		utils.McpLogf("[Error]: Authentication failed - Status: %d", statusCode)
		return createMcpErrorResponse(ErrorCodeAuthenticationFailed, "Authentication failed", "Please run 'apono login' command to authenticate")
	case http.StatusForbidden:
		utils.McpLogf("[Error]: Authorization failed - Status: %d", statusCode)
		return createMcpErrorResponse(ErrorCodeAuthorizationFailed, "Authorization failed", "Access forbidden")
	case EmptyErrorStatusCode:
		utils.McpLogf("[Error]: Failed to process request, sending error response")
		return createMcpErrorResponse(ErrorCodeInternalError, "Internal error", "Failed to process request")
	default:
		return nil
	}
}

func createMcpErrorResponse(errorCode int, message, data string) *clientapi.McpResponse {
	errorData := map[string]interface{}{
		"description": data,
	}

	mcpError := clientapi.NewMcpResponseError(int32(errorCode), message, errorData)
	response := clientapi.NewMcpResponseWithDefaults()
	response.SetJsonrpc("2.0")
	response.SetId("null")
	response.SetResult(nil)
	response.SetError(*mcpError)

	utils.McpLogf("[Error]: Creating error response - Code: %d, Message: %s, Data: %s", errorCode, message, data)
	return response
}

func parseMcpRequest(line string) (clientapi.McpRequest, string, error) {
	var requestData map[string]interface{}
	if err := json.Unmarshal([]byte(line), &requestData); err != nil {
		utils.McpLogf("[Error]: Failed to parse request JSON: %v", err)
		return clientapi.McpRequest{}, "", fmt.Errorf("failed to parse request JSON: %w", err)
	}

	isStandardRequest := requestData["id"] != nil

	if isStandardRequest {
		requestData["id"] = fmt.Sprintf("%v", requestData["id"])

		modifiedLine, err := json.Marshal(requestData)
		if err != nil {
			utils.McpLogf("[Error]: Failed to re-marshal request data: %v", err)
			return clientapi.McpRequest{}, "", fmt.Errorf("failed to re-marshal request data: %w", err)
		}

		var standardRequest clientapi.StandardMcpRequest
		if err := json.Unmarshal(modifiedLine, &standardRequest); err != nil {
			utils.McpLogf("[Error]: Failed to parse request as StandardMcpRequest: %v", err)
			return clientapi.McpRequest{}, "", fmt.Errorf("failed to parse request as StandardMcpRequest: %w", err)
		}
		mcpRequest := clientapi.StandardMcpRequestAsMcpRequest(&standardRequest)
		return mcpRequest, standardRequest.Method, nil
	} else {
		var notificationRequest clientapi.NotificationMcpRequest
		if err := json.Unmarshal([]byte(line), &notificationRequest); err != nil {
			utils.McpLogf("[Error]: Failed to parse request as NotificationMcpRequest: %v", err)
			return clientapi.McpRequest{}, "", fmt.Errorf("failed to parse request as NotificationMcpRequest: %w", err)
		}
		mcpRequest := clientapi.NotificationMcpRequestAsMcpRequest(&notificationRequest)
		return mcpRequest, notificationRequest.Method, nil
	}
}

func getClientName(mcpRequest clientapi.McpRequest) string {
	req := mcpRequest.StandardMcpRequest
	if req == nil {
		return ""
	}

	clientInfo, _ := req.GetParams()["clientInfo"].(map[string]interface{})
	name, _ := clientInfo["name"].(string)
	return name
}
