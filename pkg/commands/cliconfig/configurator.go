package cliconfig

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/commands/cliconfig/actions"
)

type Configurator struct{}

func (c *Configurator) ConfigureCommands(rootCmd *cobra.Command) error {
	configCmd := actions.Config()
	rootCmd.AddCommand(configCmd)

	configCmd.AddCommand(actions.ConfigSet())
	configCmd.AddCommand(actions.ConfigGet())

	return nil
}
