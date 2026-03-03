package actions

import (
	"fmt"

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

// requirePathArg returns a cobra.PositionalArgs that requires exactly one
// argument and produces a descriptive error message including the command name.
func requirePathArg(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("requires a secret path argument, e.g.: apono vault %s <mount>/<secret>", cmd.Name())
	}

	if len(args) > 1 {
		return fmt.Errorf("accepts 1 secret path argument but received %d", len(args))
	}

	return nil
}
