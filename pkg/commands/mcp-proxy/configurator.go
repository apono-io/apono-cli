package mcpproxy

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/commands/mcp-proxy/actions"
	"github.com/apono-io/apono-cli/pkg/groups"
)

type Configurator struct{}

func (c *Configurator) ConfigureCommands(rootCmd *cobra.Command) error {
	rootCmd.AddGroup(groups.OtherCommandsGroup)
	rootCmd.AddCommand(actions.MCPProxy())

	return nil
}

