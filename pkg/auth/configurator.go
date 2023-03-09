package auth

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/auth/commands"
)

type Configurator struct{}

func (c *Configurator) ConfigureCommands(rootCmd *cobra.Command) error {
	rootCmd.AddGroup(commands.Group)
	rootCmd.AddCommand(commands.GetProfiles())
	rootCmd.AddCommand(commands.Login())
	rootCmd.AddCommand(commands.Logout())
	rootCmd.AddCommand(commands.SetProfile())
	return nil
}
