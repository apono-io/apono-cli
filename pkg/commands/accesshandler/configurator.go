package accesshandler

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/commands/accesshandler/actions"
)

type Configurator struct{}

func (c *Configurator) ConfigureCommands(rootCmd *cobra.Command) error {
	cmd := actions.AccessHandler()
	cmd.AddCommand(actions.Register())
	cmd.AddCommand(actions.Unregister())
	rootCmd.AddCommand(cmd)
	return nil
}
