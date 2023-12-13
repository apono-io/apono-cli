package access

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/access/commands"
)

type Configurator struct{}

func (c *Configurator) ConfigureCommands(rootCmd *cobra.Command) error {
	accessCmd := commands.Access()
	accessCmd.AddGroup(commands.Group)

	rootCmd.AddCommand(accessCmd)
	rootCmd.AddGroup(commands.Group)

	accessCmd.AddCommand(commands.AccessList())
	accessCmd.AddCommand(commands.AccessDetails())
	accessCmd.AddCommand(commands.AccessReset())
	return nil
}
