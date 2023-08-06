package integrations

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/integrations/commands"
)

type Configurator struct{}

func (c *Configurator) ConfigureCommands(rootCmd *cobra.Command) error {
	rootCmd.AddGroup(commands.Group)
	rootCmd.AddCommand(commands.ListIntegrations())
	rootCmd.AddCommand(commands.ListPermissions())
	rootCmd.AddCommand(commands.ListResourceTypes())
	rootCmd.AddCommand(commands.ListResources())
	return nil
}
