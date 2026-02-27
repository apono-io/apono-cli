package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMCPServersConfig_ValidFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "mcp-servers.yaml")

	yaml := `mcp_servers:
  - id: postgres
    name: "PostgreSQL MCP"
    integration_types: ["postgresql", "postgres", "rds-postgresql"]
    command: "npx"
    args: ["-y", "@anthropic-ai/postgres-mcp-server"]
    credential_builder:
      database_url: "postgresql://{{.username}}:{{.password}}@{{.host}}:{{.port}}/{{.db_name}}?sslmode=require"
    env_mapping:
      database_url: "DATABASE_URL"
    arg_mapping:
      - database_url
`
	if err := os.WriteFile(configPath, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	registry, err := LoadMCPServersConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(registry.Servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(registry.Servers))
	}

	s := registry.Servers[0]
	if s.ID != "postgres" {
		t.Errorf("expected id 'postgres', got %q", s.ID)
	}
	if s.Command != "npx" {
		t.Errorf("expected command 'npx', got %q", s.Command)
	}
	if len(s.IntegrationTypes) != 3 {
		t.Errorf("expected 3 integration types, got %d", len(s.IntegrationTypes))
	}
	if s.CredentialBuilder["database_url"] == "" {
		t.Error("expected credential_builder.database_url to be set")
	}
	if s.EnvMapping["database_url"] != "DATABASE_URL" {
		t.Errorf("expected env_mapping database_url=DATABASE_URL, got %q", s.EnvMapping["database_url"])
	}
	if len(s.ArgMapping) != 1 || s.ArgMapping[0] != "database_url" {
		t.Errorf("expected arg_mapping [database_url], got %v", s.ArgMapping)
	}
}

func TestLoadMCPServersConfig_FileNotFound(t *testing.T) {
	_, err := LoadMCPServersConfig("/nonexistent/path.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLookupByIntegrationType(t *testing.T) {
	registry := &MCPServersConfig{
		Servers: []MCPServerDefinition{
			{
				ID:               "postgres",
				IntegrationTypes: []string{"postgresql", "postgres", "rds-postgresql"},
				Command:          "npx",
			},
		},
	}

	s, ok := registry.LookupByIntegrationType("rds-postgresql")
	if !ok {
		t.Fatal("expected to find server for rds-postgresql")
	}
	if s.ID != "postgres" {
		t.Errorf("expected postgres, got %q", s.ID)
	}

	_, ok = registry.LookupByIntegrationType("mysql")
	if ok {
		t.Fatal("expected no match for mysql")
	}
}
