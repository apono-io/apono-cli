package integration

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/integration/commands"
)

type Configurator struct{}

var rootIntegrationCmd = &cobra.Command{
	Use:     "integration",
	Aliases: []string{"integrations"},
	GroupID: commands.Group.ID,
	Short:   "Manage your integrations.",
}

func (c *Configurator) ConfigureCommands(rootCmd *cobra.Command) error {
	rootCmd.AddGroup(commands.Group)
	rootCmd.AddCommand(rootIntegrationCmd)
	rootIntegrationCmd.AddCommand(commands.List())
	rootIntegrationCmd.AddCommand(commands.Describe())
	return nil
}
