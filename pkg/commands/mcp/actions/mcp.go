package actions

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/config"
	"github.com/apono-io/apono-cli/pkg/groups"
	"github.com/apono-io/apono-cli/pkg/utils"
)

const (
	McpEndpointPath      = "/api/client/v1/mcp"
	JsonrpcVersion       = "2.0"
	ParseError           = -32700
	InternalError        = -32603
	AuthError            = -32002
	EmptyErrorStatusCode = 0
)

type McpRequest struct {
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	Meta    interface{} `json:"_meta"`
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
}

type McpResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

func MCP() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "mcp",
		Short:             "Run stdio MCP server",
		GroupID:           groups.OtherCommandsGroup.ID,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error { return nil },
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := utils.InitMcpLogFile(); err != nil {
				return fmt.Errorf("failed to initialize MCP log file: %w", err)
			}
			defer utils.CloseMcpLogFile()

			utils.McpLogf("=== Apono MCP STDIO Server Starting ===")

			endpoint, httpClient, err := createAponoMCPClient(cmd)
			if err != nil {
				return fmt.Errorf("failed to setup MCP server: %w", err)
			}

			utils.McpLogf("Ready to receive requests...")

			return runSTDIOServer(endpoint, httpClient)
		},
	}

	return cmd
}

func createAponoMCPClient(cmd *cobra.Command) (string, *http.Client, error) {
	utils.McpLogf("=== Starting Setup ===")

	sessionCfg, err := config.GetProfileByName("")
	if err != nil {
		return "", nil, fmt.Errorf("failed to get profile: %w", err)
	}

	apiURL, err := url.Parse(sessionCfg.ApiURL)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse API URL: %w", err)
	}
	apiURL.Path = McpEndpointPath

	var httpClient *http.Client
	if sessionCfg.PersonalToken != "" {
		httpClient = aponoapi.HTTPClientWithPersonalToken(sessionCfg.PersonalToken)
	} else {
		client, err := aponoapi.CreateClient(cmd.Context(), "")
		if err != nil {
			return "", nil, fmt.Errorf("failed to create API client: %w", err)
		}
		httpClient = client.APIClient.GetConfig().HTTPClient
	}

	utils.McpLogf("=== Setup Finished ===")
	return apiURL.String(), httpClient, nil
}

func runSTDIOServer(endpoint string, httpClient *http.Client) error {
	scanner := bufio.NewScanner(os.Stdin)

	utils.McpLogf("=== STDIO Server Started, waiting for input ===")

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		utils.McpLogf("Received Line: %s", line)

		var request McpRequest
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			utils.McpLogf("ERROR: Failed to parse JSON: %v", err)
			response := McpResponse{
				JSONRPC: JsonrpcVersion,
				ID:      nil,
				Error: map[string]interface{}{
					"code":    ParseError,
					"message": "Request received cannot be parsed as an MCP request",
					"data":    fmt.Sprintf("Invalid JSON: %v", err),
				},
			}
			sendResponse(response, EmptyErrorStatusCode)
			continue
		}

		utils.McpLogf("Parsed request - Method: %s, ID: %v", request.Method, request.ID)

		response, statusCode := sendMcpRequest(endpoint, httpClient, request)

		if !isNotificationsMethod(request.Method) {
			utils.McpLogf("Sending response - Status: %d, ID: %v", statusCode, response.ID)
			sendResponse(response, statusCode)
		}
	}

	if err := scanner.Err(); err != nil {
		utils.McpLogf("ERROR: Scanner error: %v", err)
		return fmt.Errorf("error reading stdin: %w", err)
	}

	return nil
}

func sendMcpRequest(endpoint string, httpClient *http.Client, request McpRequest) (McpResponse, int) {
	utils.McpLogf("Sending request to endpoint: %s", endpoint)

	requestBody, err := json.Marshal(request)
	if err != nil {
		utils.McpLogf("ERROR: Failed to marshal request: %v", err)
		return McpResponse{
			JSONRPC: JsonrpcVersion,
			ID:      request.ID,
			Error: map[string]interface{}{
				"code":    InternalError,
				"message": "An internal error occurred when trying to send the MCP request to Apono",
				"data":    "Failed to marshal request",
			},
		}, EmptyErrorStatusCode
	}

	utils.McpLogf("Request body: %s", string(requestBody))

	httpReq, err := http.NewRequestWithContext(context.Background(), "POST", endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		utils.McpLogf("ERROR: Failed to create HTTP request: %v", err)
		return McpResponse{
			JSONRPC: JsonrpcVersion,
			ID:      request.ID,
			Error: map[string]interface{}{
				"code":    InternalError,
				"message": "An internal error occurred when trying to send the MCP request to Apono",
				"data":    "Failed to create HTTP request",
			},
		}, EmptyErrorStatusCode
	}

	httpReq.Header.Set("Content-Type", "application/json")

	utils.McpLogf("Sending HTTP request...")
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		utils.McpLogf("ERROR: Failed to send HTTP request: %v", err)
		return McpResponse{
			JSONRPC: JsonrpcVersion,
			ID:      request.ID,
			Error: map[string]interface{}{
				"code":    InternalError,
				"message": "An internal error occurred when trying to send the MCP request to Apono",
				"data":    fmt.Sprintf("Failed to send HTTP request: %v", err),
			},
		}, EmptyErrorStatusCode
	}
	defer func(Body io.ReadCloser) {
		closeErr := Body.Close()
		if closeErr != nil {
			utils.McpLogf("WARNING: Failed to close response body: %v", closeErr)
		}
	}(resp.Body)

	utils.McpLogf("HTTP response status: %d", resp.StatusCode)

	if resp.StatusCode == http.StatusForbidden {
		return McpResponse{
			JSONRPC: JsonrpcVersion,
			ID:      request.ID,
			Error: map[string]interface{}{
				"code":    AuthError,
				"message": "Authentication failed. Please run 'apono login' to authenticate.",
				"data":    fmt.Sprintf("Status code %v - Invalid or expired authentication token", resp.StatusCode),
			},
		}, resp.StatusCode
	}

	// For notification methods, don't read response body - just log completion
	if isNotificationsMethod(request.Method) {
		utils.McpLogf("Notification method %s completed successfully with status %d", request.Method, resp.StatusCode)
		return McpResponse{}, resp.StatusCode
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.McpLogf("ERROR: Failed to read response body: %v", err)
		return McpResponse{
			JSONRPC: JsonrpcVersion,
			ID:      request.ID,
			Error: map[string]interface{}{
				"code":    InternalError,
				"message": "An internal error occurred when trying to send the MCP request to Apono",
				"data":    "Failed to read response body",
			},
		}, resp.StatusCode
	}

	utils.McpLogf("Response body: %s", string(responseBody))

	var response McpResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		utils.McpLogf("ERROR: Failed to unmarshal response: %v", err)
		return McpResponse{
			JSONRPC: JsonrpcVersion,
			ID:      request.ID,
			Error: map[string]interface{}{
				"code":    InternalError,
				"message": "An internal error occurred when trying to unmarshal the MCP response from the server",
				"data":    "Failed to parse response",
			},
		}, resp.StatusCode
	}

	if !isNotificationsMethod(request.Method) {
		response.ID = request.ID
	}

	return response, resp.StatusCode
}

func sendResponse(response McpResponse, statusCode int) {
	responseJSON, err := json.Marshal(response)
	if err != nil {
		if statusCode == http.StatusForbidden {
			errorResponse := McpResponse{
				JSONRPC: JsonrpcVersion,
				ID:      response.ID,
				Error: map[string]interface{}{
					"code":    AuthError,
					"message": "Authentication failed. Please run 'apono login' to authenticate.",
					"data":    fmt.Sprintf("Status code %v - Invalid or expired authentication token", statusCode),
				},
			}
			errorJSON, _ := json.Marshal(errorResponse)
			fmt.Println(string(errorJSON))
			return
		}

		errorResponse := McpResponse{
			JSONRPC: JsonrpcVersion,
			ID:      response.ID,
			Error: map[string]interface{}{
				"code":    InternalError,
				"message": "An internal error occurred when trying to send the MCP request to Apono",
				"data":    "Failed to marshal response",
			},
		}
		errorJSON, _ := json.Marshal(errorResponse)
		fmt.Println(string(errorJSON))
		return
	}

	utils.McpLogf("Sending response: %s", string(responseJSON))
	fmt.Println(string(responseJSON))
}

func isNotificationsMethod(method string) bool {
	return strings.HasPrefix(method, "notifications/")
}
