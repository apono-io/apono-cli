package registry

// MCPServerDefinition defines how to spawn and configure an MCP server
// for a given integration type. Loaded from ~/.apono/mcp-servers.yaml.
type MCPServerDefinition struct {
	ID                string            `yaml:"id"`
	Name              string            `yaml:"name"`
	IntegrationTypes  []string          `yaml:"integration_types"`
	Command           string            `yaml:"command"`
	Args              []string          `yaml:"args,omitempty"`
	CredentialBuilder map[string]string `yaml:"credential_builder,omitempty"`
	EnvMapping        map[string]string `yaml:"env_mapping,omitempty"`
	ArgMapping        []string          `yaml:"arg_mapping,omitempty"`
}

// MCPServersConfig is the top-level config loaded from mcp-servers.yaml.
type MCPServersConfig struct {
	Servers []MCPServerDefinition `yaml:"mcp_servers"`
}

// LookupByIntegrationType finds the first server definition matching
// the given Apono integration type string.
func (c *MCPServersConfig) LookupByIntegrationType(integrationType string) (MCPServerDefinition, bool) {
	for _, s := range c.Servers {
		for _, it := range s.IntegrationTypes {
			if it == integrationType {
				return s, true
			}
		}
	}
	return MCPServerDefinition{}, false
}

// LookupByID finds a server definition by its ID.
func (c *MCPServersConfig) LookupByID(id string) (MCPServerDefinition, bool) {
	for _, s := range c.Servers {
		if s.ID == id {
			return s, true
		}
	}
	return MCPServerDefinition{}, false
}
