package actions

import (
	"github.com/apono-io/apono-cli/pkg/groups"
	"github.com/spf13/cobra"
)

func Vault() *cobra.Command {
	return &cobra.Command{
		Use:     "vault",
		Short:   "Manage Apono vault secrets",
		GroupID: groups.ManagementCommandsGroup.ID,
	}
}
