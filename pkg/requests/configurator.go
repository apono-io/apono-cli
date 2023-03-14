package requests

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/requests/commands"
)

type Configurator struct{}

func (c *Configurator) ConfigureCommands(rootCmd *cobra.Command) error {
	rootCmd.AddGroup(commands.Group)
	rootCmd.AddCommand(commands.List())
	rootCmd.AddCommand(commands.New())
	return nil
}
