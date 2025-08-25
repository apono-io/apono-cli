package mcp

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/commands/mcp/actions"
	"github.com/apono-io/apono-cli/pkg/groups"
)

type Configurator struct{}

func (c *Configurator) ConfigureCommands(rootCmd *cobra.Command) error {
	rootCmd.AddGroup(groups.OtherCommandsGroup)
	rootCmd.AddCommand(actions.MCP())

	return nil
}
