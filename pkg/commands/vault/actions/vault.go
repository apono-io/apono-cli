package actions

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/groups"
	"github.com/apono-io/apono-cli/pkg/services"
)

func Vault() *cobra.Command {
	return &cobra.Command{
		Use:     "vault",
		Short:   "Manage Apono vault secrets",
		GroupID: groups.ManagementCommandsGroup.ID,
	}
}

func requirePathArg(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("requires a secret path argument, e.g.: apono vault %s <mount>/<secret>", cmd.Name())
	}

	if len(args) > 1 {
		return fmt.Errorf("accepts 1 secret path argument but received %d", len(args))
	}

	return nil
}

func vaultWriteCommand(use, short, pastTense string, requireNew bool) *cobra.Command {
	var vaultID string
	var value string

	cmd := &cobra.Command{
		Use:   use + " <path>",
		Short: short,
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

			vc, _, err := services.ResolveVaultClient(ctx, client, vaultID)
			if err != nil {
				return err
			}

			mount, secretName, err := services.ParseVaultPath(secretPath)
			if err != nil {
				return err
			}

			if requireNew {
				var exists bool
				exists, err = vc.SecretExists(ctx, mount, secretName)
				if err != nil {
					return err
				}

				if exists {
					return fmt.Errorf("secret %q already exists in mount %q; use 'apono vault update' to modify it", secretName, mount)
				}
			}

			err = vc.WriteSecret(ctx, mount, secretName, secretData)
			if err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Secret %q %s successfully\n", secretPath, pastTense)
			return err
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&vaultID, "vault-id", "", "The vault integration name or ID")
	flags.StringVar(&value, "value", "", "Secret value as JSON (e.g. '{\"key\":\"val\"}')")
	_ = cmd.MarkFlagRequired("vault-id")
	_ = cmd.MarkFlagRequired("value")

	return cmd
}
