package registry

// DefaultConfig returns the built-in MCP server definitions.
// Used as fallback when ~/.apono/mcp-servers.yaml doesn't exist.
func DefaultConfig() *MCPServersConfig {
	return &MCPServersConfig{
		Servers: []MCPServerDefinition{
			{
				ID:               "postgres",
				Name:             "PostgreSQL MCP",
				IntegrationTypes: []string{"postgresql", "postgres", "rds-postgresql", "azure-postgresql", "gcp-postgresql"},
				Command:          "npx",
				Args:             []string{"-y", "@modelcontextprotocol/server-postgres"},
				CredentialBuilder: map[string]string{
					"database_url": "postgresql://{{.username}}:{{urlEncode .password}}@{{.host}}:{{.port}}/{{.db_name}}?sslmode=require",
				},
				ArgMapping: []string{"database_url"},
			},
		},
	}
}
