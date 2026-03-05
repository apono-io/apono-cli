package actions

import (
	"github.com/spf13/cobra"
)

func VaultUpdate() *cobra.Command {
	return vaultWriteCommand("update", "Update a secret in a vault", "updated")
}
