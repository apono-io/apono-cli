package requests

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/requests/commands"
)

type Configurator struct{}

func (c *Configurator) ConfigureCommands(rootCmd *cobra.Command) error {
	requestsRootCmd := commands.Requests()
	rootCmd.AddCommand(requestsRootCmd)
	rootCmd.AddGroup(commands.Group)

	requestsRootCmd.AddCommand(commands.List())
	requestsRootCmd.AddCommand(commands.Describe())
	requestsRootCmd.AddCommand(commands.Create())
	requestsRootCmd.AddCommand(commands.AccessUnits())
	return nil
}
