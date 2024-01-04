package integrations

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/commands/integrations/actions"
)

type Configurator struct{}

func (c *Configurator) ConfigureCommands(rootCmd *cobra.Command) error {
	rootCmd.AddGroup(actions.Group)
	rootCmd.AddCommand(actions.ListIntegrations())
	rootCmd.AddCommand(actions.ListPermissions())
	rootCmd.AddCommand(actions.ListResourceTypes())
	rootCmd.AddCommand(actions.ListResources())
	rootCmd.AddCommand(actions.ListBundles())
	return nil
}
