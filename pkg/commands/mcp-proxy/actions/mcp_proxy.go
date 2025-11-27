package actions

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/commands/mcp-proxy/approval"
	"github.com/apono-io/apono-cli/pkg/commands/mcp-proxy/auditor"
	"github.com/apono-io/apono-cli/pkg/commands/mcp-proxy/config"
	"github.com/apono-io/apono-cli/pkg/commands/mcp-proxy/notifier"
	"github.com/apono-io/apono-cli/pkg/commands/mcp-proxy/transport"
	"github.com/apono-io/apono-cli/pkg/groups"
	"github.com/apono-io/apono-cli/pkg/utils"
)

const (
	debugFlagName  = "debug"
	modeFlagName   = "mode"
	configFlagName = "config"
)

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

func MCPProxy() *cobra.Command {
	var debug bool
	var mode string
	var configFile string

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

			// Load configuration
			cfg, err := config.LoadConfig(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Initialize base auditor
			baseAud, err := auditor.NewAuditor(auditor.AuditorConfig{
				Type: "file",
				// FilePath will use default if empty
			})
			if err != nil {
				return fmt.Errorf("failed to initialize auditor: %w", err)
			}
			defer baseAud.Close()

			// Initialize risk detector
			var riskConfig auditor.RiskConfig
			if cfg.Risk.Enabled {
				riskConfig = cfg.ToRiskConfig()
			} else {
				riskConfig = auditor.DefaultRiskConfig()
			}
			riskDetector := auditor.NewPatternRiskDetector(riskConfig)

			// Initialize approval workflow based on notification type
			var approvalMgr auditor.ApprovalManager
			var useApprovalFlow bool

			// Set up approval workflow based on notification type
			if cfg.Slack.Enabled {
				utils.McpLogf("Initializing Slack approval workflow...")

				// Create approval store
				approvalStore := approval.NewInMemoryApprovalStore()

				// Create Slack notifier
				slackNotifier := notifier.NewSlackNotifier(cfg.Slack.BotToken, cfg.Slack.ChannelID, cfg.Slack.UserID)

				// Create callback handler
				callbackHandler := notifier.NewCallbackHandler(cfg.Slack.SigningSecret, cfg.Slack.SkipVerification, approvalStore)

				// Start callback server in background
				go func() {
					utils.McpLogf("Starting Slack callback server on port %d", cfg.Slack.CallbackPort)
					if err := notifier.StartCallbackServer(cfg.Slack.CallbackPort, callbackHandler); err != nil {
						utils.McpLogf("Slack callback server error: %v", err)
					}
				}()

				// Create approval manager
				approvalMgr = approval.NewApprovalManager(approvalStore, slackNotifier, cfg.Slack.Timeout)
				utils.McpLogf("Slack approval workflow initialized on port %d", cfg.Slack.CallbackPort)
				useApprovalFlow = true
			}

			// Wrap with risk-aware auditor
			aud := auditor.NewRiskAwareAuditor(baseAud, riskDetector, riskConfig.BlockOnRisk, approvalMgr, useApprovalFlow)
			utils.McpLogf("Auditor with risk detection initialized successfully")

			utils.McpLogf("=== MCP Proxy Server Starting ===")

			// Determine mode: if no mode flag is provided, auto-detect from config
			// If Command is set in config, use STDIO mode; otherwise use HTTP mode
			effectiveMode := mode
			if effectiveMode == "http" && cfg.Proxy.Command != "" {
				effectiveMode = "stdio"
			}
			utils.McpLogf("Mode: %s", effectiveMode)

			if effectiveMode == "stdio" {
				// STDIO subprocess mode - use config from file
				if cfg.Proxy.Command == "" {
					return fmt.Errorf("STDIO mode requires 'proxy.command' in config file")
				}

				utils.McpLogf("Proxying to subprocess: %s (command: %s)", cfg.Proxy.Name, cfg.Proxy.Command)
				utils.McpLogf("Ready to receive requests...")

				return transport.RunSTDIOProxy(transport.STDIOProxyConfig{
					Command: cfg.Proxy.Command,
					Args:    cfg.Proxy.Args,
					Env:     cfg.Proxy.Env,
					Debug:   debug,
					Auditor: aud,
				})
			}

			// HTTP mode - use config from file
			if cfg.Proxy.Endpoint == "" {
				return fmt.Errorf("HTTP mode requires 'proxy.endpoint' in config file")
			}

			utils.McpLogf("Proxying to: %s (%s)", cfg.Proxy.Name, cfg.Proxy.Endpoint)
			utils.McpLogf("Ready to receive requests...")

			return transport.RunSTDIOServer(transport.STDIOServerConfig{
				Endpoint:        cfg.Proxy.Endpoint,
				HTTPClient:      &http.Client{},
				RequestModifier: &ProxyRequestModifier{headers: cfg.Proxy.Headers},
				Debug:           debug,
				Auditor:         aud,
			})
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&debug, debugFlagName, false, "Enable debug logging for request/response bodies")
	flags.StringVar(&mode, modeFlagName, "http", "Proxy mode: 'http' for HTTP endpoint, 'stdio' for subprocess")
	flags.StringVar(&configFile, configFlagName, "", "Path to configuration file (YAML)")

	return cmd
}
