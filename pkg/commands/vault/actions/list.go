package actions

import (
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/utils"
	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
)

func VaultList() *cobra.Command {
	format := new(utils.Format)
	var vaultID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List secrets in a vault",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

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

			// Fetch mount_name from access details JSON
			accessDetails, _, err := client.ClientAPI.AccessSessionsAPI.GetAccessSessionAccessDetails(ctx, session.Id).
				FormatType(services.JSONOutputFormat).
				Execute()
			if err != nil {
				return fmt.Errorf("failed to get session details: %w", err)
			}

			mountName, _ := accessDetails.GetJson()["mount_name"].(string)
			if mountName == "" {
				mountName = "kv"
			}

			vc, err := services.VaultLogin(creds.VaultAddress, creds.Username, creds.Password)
			if err != nil {
				return err
			}

			keys, err := vc.List(services.VaultKVMetadataPath(mountName))
			if err != nil {
				return err
			}

			if len(keys) == 0 {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "No secrets found")
				return err
			}

			switch *format {
			case utils.JSONFormat:
				paths := make([]string, len(keys))
				for i, key := range keys {
					paths[i] = mountName + "/" + key
				}
				return utils.PrintObjectsAsJSON(cmd.OutOrStdout(), paths)
			case utils.YamlFormat:
				paths := make([]string, len(keys))
				for i, key := range keys {
					paths[i] = mountName + "/" + key
				}
				return utils.PrintObjectsAsYaml(cmd.OutOrStdout(), paths)
			default:
				table := uitable.New()
				table.AddRow("SECRET PATH")
				for _, key := range keys {
					table.AddRow(mountName + "/" + key)
				}

				_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
				return err
			}
		},
	}

	flags := cmd.Flags()
	utils.AddFormatFlag(flags, format)
	flags.StringVar(&vaultID, "vault-id", "", "The vault integration ID or type/name")
	_ = cmd.MarkFlagRequired("vault-id")

	return cmd
}
