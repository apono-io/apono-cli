package transport

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
	"strings"
	"syscall"

	"github.com/apono-io/apono-cli/pkg/utils"
)

const (
	EmptyErrorStatusCode = 0
	mcpMethodInitialize  = "initialize"

	ErrorCodeInternalError = -32603
)

// RequestModifier allows customizing HTTP requests and error handling
type RequestModifier interface {
	// ModifyRequest adds headers, user-agent, etc to the HTTP request
	ModifyRequest(req *http.Request, clientName string)

	// HandleErrorResponse optionally transforms error responses based on status
	// Returns (errorResponse, handled) where handled=true means this modifier handled the error
	HandleErrorResponse(statusCode int) (errorResponse string, handled bool)
}

// STDIOServerConfig configures the STDIO server
type STDIOServerConfig struct {
	Endpoint        string
	HTTPClient      *http.Client
	RequestModifier RequestModifier
	Debug           bool
}

// InitializeClientParams represents the initialize request structure
type InitializeClientParams struct {
	Params struct {
		ClientInfo struct {
			Name string `json:"name"`
		} `json:"clientInfo"`
	} `json:"params"`
}

// RunSTDIOServer runs a generic STDIO<->HTTP MCP proxy
func RunSTDIOServer(config STDIOServerConfig) error {
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

			// Log and parse request
			var requestData map[string]interface{}
			if err := json.Unmarshal([]byte(line), &requestData); err == nil {
				if method, ok := requestData["method"].(string); ok {
					utils.McpLogf("Received request method: %q", method)
					if config.Debug {
						utils.McpLogf("[Debug]: Request body: %s", line)
					}

					// Extract client name from initialize
					if strings.ToLower(method) == mcpMethodInitialize {
						name, err := ExtractClientNameFromInitialize(requestData)
						if err != nil {
							utils.McpLogf("[Error]: Failed to extract client name: %v", err)
						} else if name != "" {
							clientName = name
							utils.McpLogf("Client name set to: %s", clientName)
						}
					}
				}
			}

			// Send request
			response, statusCode := SendHTTPRequest(
				config.Endpoint,
				config.HTTPClient,
				line,
				clientName,
				config.RequestModifier,
				config.Debug,
			)

			// Handle errors
			if statusCode == EmptyErrorStatusCode {
				utils.McpLogf("[Error]: Failed to process request, sending error response")
				errorResponse := CreateErrorResponse(ErrorCodeInternalError, "Internal error", "Failed to process request")
				fmt.Println(errorResponse)
				continue
			}

			// Output response
			PropagateResponseToStdout(response, statusCode)
		}
	}
}

// SendHTTPRequest sends an MCP request over HTTP
func SendHTTPRequest(
	endpoint string,
	httpClient *http.Client,
	request string,
	clientName string,
	modifier RequestModifier,
	debug bool,
) (string, int) {
	if debug {
		utils.McpLogf("[Debug]: Sending request to endpoint: %s", endpoint)
	}

	httpReq, err := http.NewRequestWithContext(
		context.Background(),
		"POST",
		endpoint,
		bytes.NewBuffer([]byte(request)),
	)
	if err != nil {
		utils.McpLogf("[Error]: Failed to create HTTP request: %v", err)
		return "", EmptyErrorStatusCode
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Allow custom request modifications
	if modifier != nil {
		modifier.ModifyRequest(httpReq, clientName)
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

	// Check for custom error handling
	if modifier != nil {
		if errorResp, handled := modifier.HandleErrorResponse(resp.StatusCode); handled {
			return errorResp, resp.StatusCode
		}
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.McpLogf("[Error]: Failed to read response body: %v", err)
		return "", resp.StatusCode
	}

	if debug {
		utils.McpLogf("[Debug]: Response body: %s", string(responseBody))
	}

	return string(responseBody), resp.StatusCode
}

// ExtractClientNameFromInitialize extracts client name from initialize request
func ExtractClientNameFromInitialize(requestData map[string]interface{}) (string, error) {
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

// PropagateResponseToStdout writes response to stdout
func PropagateResponseToStdout(response string, statusCode int) {
	if response != "" {
		fmt.Println(response)
		return
	}
	utils.McpLogf("No response body to send, status: %d", statusCode)
}

// CreateErrorResponse creates a JSON-RPC error response
func CreateErrorResponse(errorCode int, message, data string) string {
	return fmt.Sprintf(
		`{"jsonrpc":"2.0","id":null,"error":{"code":%d,"message":"%s","data":"%s"}}`,
		errorCode, message, data,
	)
}
