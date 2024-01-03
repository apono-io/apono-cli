package integrations

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/commands/integrations/actions"
)

type Configurator struct{}

func (c *Configurator) ConfigureCommands(rootCmd *cobra.Command) error {
	inventoryRootCmd := actions.Inventory()
	rootCmd.AddCommand(inventoryRootCmd)

	inventoryRootCmd.AddCommand(actions.ListIntegrations())
	inventoryRootCmd.AddCommand(actions.ListPermissions())
	inventoryRootCmd.AddCommand(actions.ListResourceTypes())
	inventoryRootCmd.AddCommand(actions.ListResources())
	inventoryRootCmd.AddCommand(actions.ListBundles())
	return nil
}
