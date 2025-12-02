package actions

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/config"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/spf13/cobra"
)

func AccessSaveCreds() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "save-creds <session_id>",
		Short: "Save access session credentials to a local env file",
		Long: `Save access session credentials to a local .creds file that can be sourced in your terminal.

The file will be saved in the Apono config directory (~/.config/apono-cli or equivalent).
You can then source it with: source ~/.config/apono-cli/<session_id>.creds

Example:
  apono access save-creds postgresql-local-postgres-adce68
  source ~/.config/apono-cli/postgresql-local-postgres-adce68.creds`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("missing session id")
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

			// Generate env file content
			envContent := generateEnvFileContent(sessionID, creds)

			// Save to config directory
			credsFilePath := path.Join(config.DirPath, fmt.Sprintf("%s.creds", sessionID))
			if err := os.WriteFile(credsFilePath, []byte(envContent), 0600); err != nil {
				return fmt.Errorf("failed to write credentials file: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "✓ Credentials saved to: %s\n", credsFilePath)
			fmt.Fprintf(cmd.OutOrStdout(), "\nTo use these credentials, run:\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  source %s\n", credsFilePath)

			return nil
		},
	}

	return cmd
}

func generateEnvFileContent(sessionID string, creds map[string]interface{}) string {
	var sb strings.Builder

	// Add header comment
	sb.WriteString(fmt.Sprintf("# Apono credentials for session: %s\n", sessionID))
	sb.WriteString(fmt.Sprintf("# Generated at: %s\n", os.Getenv("DATE")))
	sb.WriteString("# Source this file to load credentials into your environment:\n")
	sb.WriteString(fmt.Sprintf("#   source %s\n\n", path.Join(config.DirPath, fmt.Sprintf("%s.creds", sessionID))))

	// Convert all credential fields to uppercase env var format
	for key, value := range creds {
		if value == nil {
			continue
		}

		// Convert key to uppercase env var format (e.g., db_name -> DB_NAME)
		envKey := strings.ToUpper(key)

		// Format the value as a string
		var envValue string
		switch v := value.(type) {
		case string:
			envValue = v
		case float64:
			envValue = fmt.Sprintf("%.0f", v)
		case bool:
			envValue = fmt.Sprintf("%t", v)
		default:
			envValue = fmt.Sprintf("%v", v)
		}

		// Write env var line (escape double quotes in value)
		envValue = strings.ReplaceAll(envValue, `"`, `\"`)
		sb.WriteString(fmt.Sprintf("export %s=\"%s\"\n", envKey, envValue))
	}

	// Add PostgreSQL-specific convenience vars if this looks like a DB connection
	if _, hasHost := creds["host"]; hasHost {
		if _, hasPort := creds["port"]; hasPort {
			if _, hasDbName := creds["db_name"]; hasDbName {
				sb.WriteString("\n# PostgreSQL connection string\n")
				sb.WriteString("export DATABASE_URL=\"postgresql://${USERNAME}:${PASSWORD}@${HOST}:${PORT}/${DB_NAME}\"\n")
				sb.WriteString("export PGHOST=\"${HOST}\"\n")
				sb.WriteString("export PGPORT=\"${PORT}\"\n")
				sb.WriteString("export PGUSER=\"${USERNAME}\"\n")
				sb.WriteString("export PGPASSWORD=\"${PASSWORD}\"\n")
				sb.WriteString("export PGDATABASE=\"${DB_NAME}\"\n")
			}
		}
	}

	return sb.String()
}
