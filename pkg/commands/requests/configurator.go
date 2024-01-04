package requests

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/commands/requests/actions"
)

type Configurator struct{}

func (c *Configurator) ConfigureCommands(rootCmd *cobra.Command) error {
	requestsRootCmd := actions.Requests()
	rootCmd.AddCommand(requestsRootCmd)

	requestsRootCmd.AddCommand(actions.List())
	requestsRootCmd.AddCommand(actions.Describe())
	requestsRootCmd.AddCommand(actions.Create())
	requestsRootCmd.AddCommand(actions.AccessUnits())
	requestsRootCmd.AddCommand(actions.Revoke())
	return nil
}
