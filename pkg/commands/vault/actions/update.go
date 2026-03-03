package actions

import (
	"encoding/json"
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/spf13/cobra"
)

func VaultUpdate() *cobra.Command {
	var vaultID string
	var value string

	cmd := &cobra.Command{
		Use:   "update <path>",
		Short: "Update a secret in a vault",
		Args:  requirePathArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			secretPath := args[0]

			var secretData map[string]interface{}
			if err := json.Unmarshal([]byte(value), &secretData); err != nil {
				return fmt.Errorf("invalid JSON value: %w", err)
			}

			client, err := aponoapi.GetClient(ctx)
			if err != nil {
				return err
			}

			vc, _, err := services.ResolveVaultClient(ctx, client, vaultID, services.VaultManagementSessionType)
			if err != nil {
				return err
			}

			mount, secretName, err := services.ParseVaultPath(secretPath)
			if err != nil {
				return err
			}

			if err := vc.WriteSecret(ctx, mount, secretName, secretData); err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Secret %q updated successfully\n", secretPath)
			return err
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&vaultID, "vault-id", "", "The vault integration ID or type/name")
	flags.StringVar(&value, "value", "", "Secret value as JSON (e.g. '{\"key\":\"val\"}')")
	_ = cmd.MarkFlagRequired("vault-id")
	_ = cmd.MarkFlagRequired("value")

	return cmd
}
