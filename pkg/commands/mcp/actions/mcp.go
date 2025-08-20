package actions

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/config"
	"github.com/apono-io/apono-cli/pkg/groups"
	"github.com/spf13/cobra"
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
	JsonRpc string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
}

type McpResponse struct {
	JsonRpc string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

func MCP() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "mcp",
		Short:             "Run stdio MCP proxy server",
		GroupID:           groups.OtherCommandsGroup.ID,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error { return nil },
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("=== Apono MCP STDIO Server Starting ===")

			endpoint, httpClient, err := setupMCPServer(cmd)
			if err != nil {
				return fmt.Errorf("failed to setup MCP server: %w", err)
			}

			fmt.Println("Ready to receive requests...")

			return runSTDIOServer(endpoint, httpClient)
		},
	}

	return cmd
}

func setupMCPServer(cmd *cobra.Command) (string, *http.Client, error) {
	fmt.Println("=== Starting Setup ===")

	// Get the current profile configuration
	sessionCfg, err := config.GetProfileByName("")
	if err != nil {
		return "", nil, fmt.Errorf("failed to get profile: %w", err)
	}

	apiURL, err := url.Parse(sessionCfg.ApiURL)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse API URL: %w", err)
	}
	apiURL.Path = McpEndpointPath

	// Create HTTP client with authentication
	var httpClient *http.Client
	if sessionCfg.PersonalToken != "" {
		httpClient = aponoapi.HTTPClientWithPersonalToken(sessionCfg.PersonalToken)
	} else {
		// For OAuth tokens, we need to create a proper client
		client, err := aponoapi.CreateClient(cmd.Context(), "")
		if err != nil {
			return "", nil, fmt.Errorf("failed to create API client: %w", err)
		}
		httpClient = client.APIClient.GetConfig().HTTPClient
	}

	fmt.Println("=== Setup Finished ===")
	return apiURL.String(), httpClient, nil
}

func runSTDIOServer(endpoint string, httpClient *http.Client) error {
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fmt.Printf("[DEBUG] Received Line: %s\n", line)

		var request McpRequest
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			response := McpResponse{
				JsonRpc: JsonrpcVersion,
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

		response, statusCode, err := sendMcpRequest(endpoint, httpClient, request)
		if err != nil {
			response = McpResponse{
				JsonRpc: JsonrpcVersion,
				ID:      request.ID,
				Error: map[string]interface{}{
					"code":    InternalError,
					"message": "An internal error occurred when trying to send the MCP request to Apono",
					"data":    fmt.Sprintf("Failed to send to Apono API: %v", err),
				},
			}
		}

		sendResponse(response, statusCode)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stdin: %w", err)
	}

	return nil
}

func sendMcpRequest(endpoint string, httpClient *http.Client, request McpRequest) (McpResponse, int, error) {
	requestBody, err := json.Marshal(request)
	if err != nil {
		log.Printf("Failed to marshal request: %v", err)
		return McpResponse{
			JsonRpc: JsonrpcVersion,
			ID:      request.ID,
			Error: map[string]interface{}{
				"code":    InternalError,
				"message": "An internal error occurred when trying to send the MCP request to Apono",
				"data":    "Failed to marshal request",
			},
		}, EmptyErrorStatusCode, nil
	}

	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		log.Printf("Failed to create HTTP request: %v", err)
		return McpResponse{
			JsonRpc: JsonrpcVersion,
			ID:      request.ID,
			Error: map[string]interface{}{
				"code":    InternalError,
				"message": "An internal error occurred when trying to send the MCP request to Apono",
				"data":    "Failed to create HTTP request",
			},
		}, EmptyErrorStatusCode, nil
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		log.Printf("Failed to send HTTP request: %v", err)
		return McpResponse{
			JsonRpc: JsonrpcVersion,
			ID:      request.ID,
			Error: map[string]interface{}{
				"code":    InternalError,
				"message": "An internal error occurred when trying to send the MCP request to Apono",
				"data":    "Failed to send HTTP request",
			},
		}, EmptyErrorStatusCode, nil
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		return McpResponse{
			JsonRpc: JsonrpcVersion,
			ID:      request.ID,
			Error: map[string]interface{}{
				"code":    InternalError,
				"message": "An internal error occurred when trying to send the MCP request to Apono",
				"data":    "Failed to read response body",
			},
		}, resp.StatusCode, nil
	}

	var response McpResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		log.Printf("Failed to unmarshal response: %v", err)
		return McpResponse{
			JsonRpc: JsonrpcVersion,
			ID:      request.ID,
			Error: map[string]interface{}{
				"code":    InternalError,
				"message": "An internal error occurred when trying to unmarshal the MCP response from the server",
				"data":    "Failed to parse response",
			},
		}, resp.StatusCode, nil
	}

	return response, resp.StatusCode, nil
}

func sendResponse(response McpResponse, statusCode int) {
	responseJSON, err := json.Marshal(response)
	if err != nil {
		if statusCode == http.StatusForbidden {
			errorResponse := McpResponse{
				JsonRpc: JsonrpcVersion,
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
			JsonRpc: JsonrpcVersion,
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

	fmt.Println(string(responseJSON))
}
