package registry

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadMCPServersConfig reads and parses an mcp-servers.yaml file.
func LoadMCPServersConfig(path string) (*MCPServersConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP servers config: %w", err)
	}

	var config MCPServersConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse MCP servers config: %w", err)
	}

	return &config, nil
}
