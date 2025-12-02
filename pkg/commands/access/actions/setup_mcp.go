package actions

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/config"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/spf13/cobra"
)

type MCPServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

type ClaudeConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

func AccessSetupMCP() *cobra.Command {
	var autoSource bool
	var clientType string
	var readOnly bool

	cmd := &cobra.Command{
		Use:   "setup-mcp <session_id>",
		Short: "Setup MCP server config for AI clients (Claude Desktop, Claude Code, Cursor) using session credentials",
		Long: `Setup MCP server configuration for AI clients using session credentials.

This command will:
1. Save the credentials to a .creds file
2. Generate or update the MCP server configuration for the specified client
3. Optionally provide instructions to source the credentials

Supported clients:
  - claude-desktop: Claude Desktop app (macOS: ~/Library/Application Support/Claude/claude_desktop_config.json)
  - claude-code: Claude Code CLI (~/.config/claude/claude_code_config.json)
  - cursor: Cursor IDE (~/.cursor/mcp.json)

MCP Server Modes:
  - Read-write (default): Uses @henkey/postgres-mcp-server (supports INSERT/UPDATE/DELETE/UPSERT)
  - Read-only (--read-only): Uses @modelcontextprotocol/server-postgres (read-only for safety)

Note: Read-write is the default since Apono provides access control at the credential level.

Examples:
  apono access setup-mcp postgresql-local-postgres-adce68 --client cursor
  apono access setup-mcp postgresql-local-postgres-adce68 --client claude-desktop
  apono access setup-mcp postgresql-local-postgres-adce68 --client cursor --read-only`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("missing session id")
			}

			// Validate client type
			if clientType == "" {
				return fmt.Errorf("--client flag is required. Supported clients: claude-desktop, claude-code, cursor")
			}
			if clientType != "claude-desktop" && clientType != "claude-code" && clientType != "cursor" {
				return fmt.Errorf("invalid client type: %s. Supported clients: claude-desktop, claude-code, cursor", clientType)
			}

			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			sessionID := args[0]

			// Get session details in JSON format
			accessDetails, _, err := services.GetSessionDetails(cmd.Context(), client, sessionID, services.JSONOutputFormat)
			if err != nil {
				return fmt.Errorf("failed to get session details: %w", err)
			}

			// Parse JSON credentials
			var creds map[string]interface{}
			if err := json.Unmarshal([]byte(accessDetails), &creds); err != nil {
				return fmt.Errorf("failed to parse credentials JSON: %w", err)
			}

			// Save credentials file
			credsFilePath := path.Join(config.DirPath, fmt.Sprintf("%s.creds", sessionID))
			envContent := generateEnvFileContent(sessionID, creds)
			if err := os.WriteFile(credsFilePath, []byte(envContent), 0600); err != nil {
				return fmt.Errorf("failed to write credentials file: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "✓ Credentials saved to: %s\n", credsFilePath)

			// Determine database type and setup MCP config
			if isPostgreSQLSession(creds) {
				// Pass readWrite as !readOnly (default is read-write)
				if err := setupPostgreSQLMCP(cmd, sessionID, creds, credsFilePath, clientType, !readOnly); err != nil {
					return err
				}
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "\n⚠ Unknown database type. MCP setup only supports PostgreSQL currently.\n")
				fmt.Fprintf(cmd.OutOrStdout(), "You can still source the credentials file:\n")
				fmt.Fprintf(cmd.OutOrStdout(), "  source %s\n", credsFilePath)
				return nil
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&autoSource, "auto-source", false, "Automatically source the credentials file in shell config")
	cmd.Flags().StringVar(&clientType, "client", "", "AI client type (claude-desktop, claude-code, cursor) [required]")
	cmd.Flags().BoolVar(&readOnly, "read-only", false, "Enable read-only mode (no INSERT/UPDATE/DELETE). Default is read-write since Apono controls access.")
	cmd.MarkFlagRequired("client")

	return cmd
}

func isPostgreSQLSession(creds map[string]interface{}) bool {
	_, hasHost := creds["host"]
	_, hasPort := creds["port"]
	_, hasDbName := creds["db_name"]
	_, hasUsername := creds["username"]
	_, hasPassword := creds["password"]

	return hasHost && hasPort && hasDbName && hasUsername && hasPassword
}

func setupPostgreSQLMCP(cmd *cobra.Command, sessionID string, creds map[string]interface{}, credsFilePath string, clientType string, readWrite bool) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Determine config path based on client type
	var configDir, configPath string
	var clientName string

	switch clientType {
	case "claude-desktop":
		configDir = filepath.Join(homeDir, "Library", "Application Support", "Claude")
		configPath = filepath.Join(configDir, "claude_desktop_config.json")
		clientName = "Claude Desktop"
	case "claude-code":
		configDir = filepath.Join(homeDir, ".config", "claude")
		configPath = filepath.Join(configDir, "claude_code_config.json")
		clientName = "Claude Code"
	case "cursor":
		configDir = filepath.Join(homeDir, ".cursor")
		configPath = filepath.Join(configDir, "mcp.json")
		clientName = "Cursor"
	default:
		return fmt.Errorf("unsupported client type: %s", clientType)
	}

	// Read existing config or create new one
	var config ClaudeConfig
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse existing %s config: %w", clientName, err)
		}
	} else {
		config = ClaudeConfig{
			MCPServers: make(map[string]MCPServerConfig),
		}
	}

	// Create MCP server config for PostgreSQL
	serverName := fmt.Sprintf("postgres-%s", sessionID)

	// Build connection string from credentials
	host := fmt.Sprintf("%v", creds["host"])
	port := fmt.Sprintf("%v", creds["port"])
	dbName := fmt.Sprintf("%v", creds["db_name"])
	username := fmt.Sprintf("%v", creds["username"])
	password := fmt.Sprintf("%v", creds["password"])

	// URL-encode username and password to handle special characters
	encodedUsername := url.QueryEscape(username)
	encodedPassword := url.QueryEscape(password)

	connectionString := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", encodedUsername, encodedPassword, host, port, dbName)

	// Choose MCP server package based on read-write flag
	var mcpPackage string
	var mcpArgs []string
	var accessMode string
	if readWrite {
		mcpPackage = "@henkey/postgres-mcp-server"
		mcpArgs = []string{"-y", mcpPackage, "--connection-string", connectionString}
		accessMode = "read-write (INSERT/UPDATE/DELETE enabled)"
	} else {
		mcpPackage = "@modelcontextprotocol/server-postgres"
		mcpArgs = []string{"-y", mcpPackage, connectionString}
		accessMode = "read-only (safe for production)"
	}

	config.MCPServers[serverName] = MCPServerConfig{
		Command: "npx",
		Args:    mcpArgs,
	}

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create %s config directory: %w", clientName, err)
	}

	// Write updated config
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal %s config: %w", clientName, err)
	}

	if err := os.WriteFile(configPath, configData, 0600); err != nil {
		return fmt.Errorf("failed to write %s config: %w", clientName, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ MCP server config added to %s: %s\n", clientName, configPath)
	fmt.Fprintf(cmd.OutOrStdout(), "\nMCP Server Name: %s\n", serverName)
	fmt.Fprintf(cmd.OutOrStdout(), "MCP Package: %s\n", mcpPackage)
	fmt.Fprintf(cmd.OutOrStdout(), "Access Mode: %s\n", accessMode)

	if readWrite {
		fmt.Fprintf(cmd.OutOrStdout(), "\nℹ️  Read-write mode enabled. AI can execute INSERT/UPDATE/DELETE operations.\n")
		fmt.Fprintf(cmd.OutOrStdout(), "   Access control is managed by Apono at the credential level.\n")
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n📝 Next steps:\n")

	switch clientType {
	case "claude-desktop":
		fmt.Fprintf(cmd.OutOrStdout(), "1. Restart Claude Desktop app to load the new MCP server\n")
		fmt.Fprintf(cmd.OutOrStdout(), "2. The PostgreSQL MCP server will be available to query your database\n")
	case "claude-code":
		fmt.Fprintf(cmd.OutOrStdout(), "1. Restart Claude Code or reload the MCP configuration\n")
		fmt.Fprintf(cmd.OutOrStdout(), "2. The PostgreSQL MCP server will be available to query your database\n")
		fmt.Fprintf(cmd.OutOrStdout(), "3. Use 'claude mcp list' to verify the server is loaded\n")
	case "cursor":
		fmt.Fprintf(cmd.OutOrStdout(), "1. Restart Cursor IDE to load the new MCP server\n")
		fmt.Fprintf(cmd.OutOrStdout(), "2. Open Settings > Developer > Edit Config > MCP Tools to verify\n")
		fmt.Fprintf(cmd.OutOrStdout(), "3. The PostgreSQL MCP server will appear in Available Tools\n")
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nAlternatively, you can source the credentials in your terminal:\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  source %s\n", credsFilePath)

	return nil
}
