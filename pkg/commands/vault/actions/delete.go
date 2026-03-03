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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			secretPath := args[0]

			client, err := aponoapi.GetClient(ctx)
			if err != nil {
				return err
			}

			integration, err := services.GetIntegrationByIDOrByTypeAndName(ctx, client, vaultID)
			if err != nil {
				return fmt.Errorf("vault %q not found", vaultID)
			}

			session, err := services.FindVaultSession(ctx, client, integration.Id, services.VaultManagementSessionType)
			if err != nil {
				return err
			}

			if session == nil {
				return fmt.Errorf("no active management access found for vault %q, create a new request by running: apono request create", vaultID)
			}

			creds, err := services.ResolveVaultCredentials(ctx, client, integration.Id, session)
			if err != nil {
				return err
			}

			vc, err := services.VaultLogin(creds.VaultAddress, creds.Username, creds.Password)
			if err != nil {
				return err
			}

			mount, secretName, err := services.ParseVaultPath(secretPath)
			if err != nil {
				return err
			}

			if err := vc.Delete(services.VaultKVDataPath(mount, secretName)); err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Secret %q deleted successfully\n", secretPath)
			return err
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&vaultID, "vault-id", "", "The vault integration ID or type/name")
	_ = cmd.MarkFlagRequired("vault-id")

	return cmd
}
