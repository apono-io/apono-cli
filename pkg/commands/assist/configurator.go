package assist

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/commands/assist/actions"
)

type Configurator struct{}

func (c *Configurator) ConfigureCommands(rootCmd *cobra.Command) error {
	rootCmd.AddCommand(actions.Assist())
	return nil
}
