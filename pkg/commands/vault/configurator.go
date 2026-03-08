package vault

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/commands/vault/actions"
)

type Configurator struct{}

func (c *Configurator) ConfigureCommands(rootCmd *cobra.Command) error {
	vaultCmd := actions.Vault()
	rootCmd.AddCommand(vaultCmd)

	vaultCmd.AddCommand(actions.VaultFetch())
	vaultCmd.AddCommand(actions.VaultList())
	vaultCmd.AddCommand(actions.VaultCreate())
	vaultCmd.AddCommand(actions.VaultUpdate())
	vaultCmd.AddCommand(actions.VaultDelete())

	return nil
}
