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
	EmptyErrorStatusCode = 0
	debugFlagName        = "debug"

	ErrorCodeAuthenticationFailed = -32001
	ErrorCodeAuthorizationFailed  = -32003
	ErrorCodeInternalError        = -32603
)

func MCP() *cobra.Command {
	var debug bool

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

			return runSTDIOServer(endpoint, httpClient, debug)
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&debug, debugFlagName, false, "Enable debug logging for request/response bodies")

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

func runSTDIOServer(endpoint string, httpClient *http.Client, debug bool) error {
	scanner := bufio.NewScanner(os.Stdin)

	utils.McpLogf("=== STDIO Server Started, waiting for input ===")

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var requestData map[string]interface{}
		if err := json.Unmarshal([]byte(line), &requestData); err == nil {
			if method, ok := requestData["method"].(string); ok {
				utils.McpLogf("Received request method: \"%s\"", method)
				if debug {
					utils.McpLogf("[Debug]: Request body: %s", line)
				}
			}
		}

		response, statusCode := sendMcpRequest(endpoint, httpClient, line, debug)

		if statusCode == EmptyErrorStatusCode {
			utils.McpLogf("ERROR: Failed to process request, sending error response")
			errorResponse := createErrorResponse(ErrorCodeInternalError, "Internal error", "Failed to process request")
			fmt.Println(errorResponse)
			continue
		}

		propagateResponseToStdout(response, statusCode)
	}

	if err := scanner.Err(); err != nil {
		utils.McpLogf("ERROR: Scanner error: %v", err)
		return fmt.Errorf("error reading stdin: %w", err)
	}

	return nil
}

func sendMcpRequest(endpoint string, httpClient *http.Client, request string, debug bool) (string, int) {
	if debug {
		utils.McpLogf("[Debug]: Sending request to endpoint: %s", endpoint)
	}

	httpReq, err := http.NewRequestWithContext(context.Background(), "POST", endpoint, bytes.NewBuffer([]byte(request)))
	if err != nil {
		utils.McpLogf("ERROR: Failed to create HTTP request: %v", err)
		return "", EmptyErrorStatusCode
	}

	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		utils.McpLogf("ERROR: Failed to send HTTP request: %v", err)
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
		utils.McpLogf("ERROR: Authentication failed - Status: %d", resp.StatusCode)
		errorResponse := createErrorResponse(ErrorCodeAuthenticationFailed, "Authentication failed", "Please run 'apono login' command to authenticate")
		return errorResponse, resp.StatusCode
	}

	if resp.StatusCode == http.StatusForbidden {
		utils.McpLogf("ERROR: Authorization failed - Status: %d", resp.StatusCode)
		errorResponse := createErrorResponse(ErrorCodeAuthorizationFailed, "Authorization failed", "Access forbidden")
		return errorResponse, resp.StatusCode
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.McpLogf("ERROR: Failed to read response body: %v", err)
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
