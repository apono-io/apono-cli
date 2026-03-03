package actions

import (
	"encoding/json"
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/spf13/cobra"
)

func VaultCreate() *cobra.Command {
	var vaultID string
	var value string

	cmd := &cobra.Command{
		Use:   "create <path>",
		Short: "Create a secret in a vault",
		Args:  cobra.ExactArgs(1),
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

			if err := vc.Write(services.VaultKVDataPath(mount, secretName), secretData); err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Secret %q created successfully\n", secretPath)
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
