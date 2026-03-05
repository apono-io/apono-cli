package actions

import (
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/spf13/cobra"
)

func VaultDelete() *cobra.Command {
	var vaultID string

	cmd := &cobra.Command{
		Use:   "delete <path>",
		Short: "Delete a secret from a vault",
		Args:  requirePathArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			secretPath := args[0]

			client, err := aponoapi.GetClient(ctx)
			if err != nil {
				return err
			}

			vc, _, err := services.ResolveVaultClient(ctx, client, vaultID)
			if err != nil {
				return err
			}

			mount, secretName, err := services.ParseVaultPath(secretPath)
			if err != nil {
				return err
			}

			if err := vc.DeleteSecret(ctx, mount, secretName); err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Secret %q deleted successfully\n", secretPath)
			return err
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&vaultID, "vault-id", "", "The vault integration name or ID")
	_ = cmd.MarkFlagRequired("vault-id")

	return cmd
}
