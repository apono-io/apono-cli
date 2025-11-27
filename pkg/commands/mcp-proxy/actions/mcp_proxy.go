package actions

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/commands/mcp-proxy/transport"
	"github.com/apono-io/apono-cli/pkg/groups"
	"github.com/apono-io/apono-cli/pkg/utils"
)

const (
	debugFlagName = "debug"
	modeFlagName  = "mode"
)

// ProxyConfig represents downstream MCP configuration
type ProxyConfig struct {
	Name     string
	Endpoint string
	Headers  map[string]string
	// Fields for subprocess mode
	Command string
	Args    []string
	Env     map[string]string
}

// ProxyRequestModifier implements transport.RequestModifier for generic proxying
type ProxyRequestModifier struct {
	headers map[string]string
}

func (m *ProxyRequestModifier) ModifyRequest(req *http.Request, clientName string) {
	for key, value := range m.headers {
		req.Header.Set(key, value)
	}
}

func (m *ProxyRequestModifier) HandleErrorResponse(statusCode int) (string, bool) {
	// Generic proxy doesn't do special error handling
	return "", false
}

// getHardcodedProxyConfig returns the MVP hardcoded configuration for HTTP mode
func getHardcodedProxyConfig() ProxyConfig {
	return ProxyConfig{
		Name:     "Apono Proxy MCP",
		Endpoint: "http://localhost:3000/mcp",
		Headers: map[string]string{
			"Authorization": "Bearer ABC",
		},
	}
}

// getHardcodedSTDIOConfig returns the hardcoded configuration for STDIO subprocess mode
func getHardcodedSTDIOConfig() ProxyConfig {
	return ProxyConfig{
		Name:    "Postgres MCP",
		Command: "docker",
		Args: []string{
			"run",
			"-i",
			"--rm",
			"-e",
			"DATABASE_URI",
			"crystaldba/postgres-mcp",
			"--access-mode=unrestricted",
		},
		Env: map[string]string{
			"DATABASE_URI": "postgresql://postgres:postgres@localhost:5432/agentic-poc",
		},
	}
}

func MCPProxy() *cobra.Command {
	var debug bool
	var mode string

	cmd := &cobra.Command{
		Use:               "mcp-proxy",
		Short:             "Run a generic MCP proxy server",
		GroupID:           groups.OtherCommandsGroup.ID,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error { return nil },
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := utils.InitMcpLogFile("mcp_proxy_logging.log"); err != nil {
				return fmt.Errorf("failed to initialize MCP log file: %w", err)
			}
			defer utils.CloseMcpLogFile()

			utils.McpLogf("=== MCP Proxy Server Starting ===")
			utils.McpLogf("Mode: %s", mode)

			if mode == "stdio" {
				// STDIO subprocess mode
				config := getHardcodedSTDIOConfig()
				utils.McpLogf("Proxying to subprocess: %s (command: %s)", config.Name, config.Command)
				utils.McpLogf("Ready to receive requests...")

				return transport.RunSTDIOProxy(transport.STDIOProxyConfig{
					Command: config.Command,
					Args:    config.Args,
					Env:     config.Env,
					Debug:   debug,
				})
			}

			// HTTP mode (default)
			config := getHardcodedProxyConfig()
			utils.McpLogf("Proxying to: %s (%s)", config.Name, config.Endpoint)
			utils.McpLogf("Ready to receive requests...")

			return transport.RunSTDIOServer(transport.STDIOServerConfig{
				Endpoint:        config.Endpoint,
				HTTPClient:      &http.Client{},
				RequestModifier: &ProxyRequestModifier{headers: config.Headers},
				Debug:           debug,
			})
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&debug, debugFlagName, false, "Enable debug logging for request/response bodies")
	flags.StringVar(&mode, modeFlagName, "http", "Proxy mode: 'http' for HTTP endpoint, 'stdio' for subprocess")

	return cmd
}
