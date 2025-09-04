package actions

import (
	"bufio"
	"encoding/json"
	"fmt"
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

			mcpClient, err := createAponoMCPClient(cmd)
			if err != nil {
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

func createAponoMCPClient(cmd *cobra.Command) (clientapi.ApiHandleMcpMethodRequest, error) {
	utils.McpLogf("=== Starting Setup ===")

	client, err := aponoapi.GetClient(cmd.Context())
	if err != nil {
		return clientapi.ApiHandleMcpMethodRequest{}, fmt.Errorf("failed to create Apono client: %w", err)
	}

	mcpClient := client.ClientAPI.MCPServerAPI.HandleMcpMethod(cmd.Context())
	utils.McpLogf("=== Setup Finished ===")
	return mcpClient, nil
}

func runSTDIOServer(mcpClient clientapi.ApiHandleMcpMethodRequest, debug bool) error {
	scanner := bufio.NewScanner(os.Stdin)

	utils.McpLogf("=== STDIO Server Started, waiting for input ===")

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		mcpRequest, method, err := parseMcpRequest(line)
		if err != nil {
			utils.McpLogf("[Error]: %v", err)
			errorResponse := createErrorResponse(ErrorCodeInternalError, "Internal error", "Failed to parse request")
			fmt.Println(errorResponse)
			continue
		}

		utils.McpLogf("Received request method: \"%s\"", method)
		if debug {
			utils.McpLogf("[Debug]: Request body: %s", line)
		}

		response, httpResponse, err := mcpClient.McpRequest(mcpRequest).Execute()
		if err != nil {
			utils.McpLogf("[Error]: MCP client request failed: %v", err)
			errorResponse := createErrorResponse(ErrorCodeInternalError, "Internal error", "Failed to process request")
			fmt.Println(errorResponse)
			continue
		}
		statusCode := httpResponse.StatusCode

		if statusCode == EmptyErrorStatusCode {
			utils.McpLogf("[Error]: Failed to process request, sending error response")
			errorResponse := createErrorResponse(ErrorCodeInternalError, "Internal error", "Failed to process request")
			fmt.Println(errorResponse)
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

func createErrorResponse(errorCode int, message, data string) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":null,"error":{"code":%d,"message":"%s","data":"%s"}}`, errorCode, message, data)
}

func parseMcpRequest(line string) (clientapi.McpRequest, string, error) {
	var requestData map[string]interface{}
	if err := json.Unmarshal([]byte(line), &requestData); err != nil {
		return clientapi.McpRequest{}, "", fmt.Errorf("failed to parse request JSON: %w", err)
	}

	isStandardRequest := requestData["id"] != nil

	if isStandardRequest {
		var standardRequest clientapi.StandardMcpRequest
		if err := json.Unmarshal([]byte(line), &standardRequest); err != nil {
			return clientapi.McpRequest{}, "", fmt.Errorf("failed to parse request as StandardMcpRequest: %w", err)
		}
		mcpRequest := clientapi.StandardMcpRequestAsMcpRequest(&standardRequest)
		return mcpRequest, standardRequest.Method, nil
	} else {
		var notificationRequest clientapi.NotificationMcpRequest
		if err := json.Unmarshal([]byte(line), &notificationRequest); err != nil {
			return clientapi.McpRequest{}, "", fmt.Errorf("failed to parse request as NotificationMcpRequest: %w", err)
		}
		mcpRequest := clientapi.NotificationMcpRequestAsMcpRequest(&notificationRequest)
		return mcpRequest, notificationRequest.Method, nil
	}
}
