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

			vc, creds, err := services.ResolveVaultClient(ctx, client, vaultID, services.VaultManagementSessionType)
			if err != nil {
				return err
			}

			mountName := creds.MountName
			if mountName == "" {
				mountName = "kv"
			}

			keys, err := vc.List(services.VaultKVMetadataPath(mountName))
			if err != nil {
				return err
			}

			if len(keys) == 0 {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "No secrets found")
				return err
			}

			paths := make([]string, len(keys))
			for i, key := range keys {
				paths[i] = mountName + "/" + key
			}

			switch *format {
			case utils.JSONFormat:
				return utils.PrintObjectsAsJSON(cmd.OutOrStdout(), paths)
			case utils.YamlFormat:
				return utils.PrintObjectsAsYaml(cmd.OutOrStdout(), paths)
			default:
				table := uitable.New()
				table.AddRow("SECRET PATH")
				for _, p := range paths {
					table.AddRow(p)
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
