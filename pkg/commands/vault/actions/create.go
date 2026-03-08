package actions

import (
	"github.com/spf13/cobra"
)

func VaultCreate() *cobra.Command {
	return vaultWriteCommand("create", "Create a secret in a vault", "created", true)
}
