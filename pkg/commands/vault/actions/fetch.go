package actions

import (
	"fmt"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/utils"
)

func VaultFetch() *cobra.Command {
	format := new(utils.Format)
	var vaultID string

	cmd := &cobra.Command{
		Use:   "fetch <path>",
		Short: "Fetch a secret from a vault",
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

			result, err := vc.ReadSecret(ctx, mount, secretName)
			if err != nil {
				return err
			}

			switch *format {
			case utils.JSONFormat:
				return utils.PrintObjectsAsJSON(cmd.OutOrStdout(), result)
			case utils.YamlFormat:
				return utils.PrintObjectsAsYaml(cmd.OutOrStdout(), result)
			default:
				table := uitable.New()
				table.AddRow("KEY", "VALUE")
				for k, v := range result {
					table.AddRow(k, fmt.Sprintf("%v", v))
				}

				_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
				return err
			}
		},
	}

	flags := cmd.Flags()
	utils.AddFormatFlag(flags, format)
	flags.StringVar(&vaultID, "vault-id", "", "The vault integration name or ID")
	_ = cmd.MarkFlagRequired("vault-id")

	return cmd
}
