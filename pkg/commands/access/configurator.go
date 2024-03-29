package access

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/commands/access/actions"
)

type Configurator struct{}

func (c *Configurator) ConfigureCommands(rootCmd *cobra.Command) error {
	accessCmd := actions.Access()
	rootCmd.AddCommand(accessCmd)

	accessCmd.AddCommand(actions.AccessList())
	accessCmd.AddCommand(actions.AccessDetails())
	accessCmd.AddCommand(actions.AccessReset())
	return nil
}
