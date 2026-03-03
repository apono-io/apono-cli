package actions

import (
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/utils"
	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
)

func VaultFetch() *cobra.Command {
	format := new(utils.Format)
	var vaultID string

	cmd := &cobra.Command{
		Use:   "fetch <path>",
		Short: "Fetch a secret from a vault",
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

			session, err := services.FindVaultSession(ctx, client, integration.Id, services.VaultSecretSessionType)
			if err != nil {
				return err
			}

			if session == nil {
				return fmt.Errorf("no active access found for vault %q, create a new request by running: apono request create", vaultID)
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

			result, err := vc.Read(services.VaultKVDataPath(mount, secretName))
			if err != nil {
				return err
			}

			switch *format {
			case utils.JSONFormat:
				return utils.PrintObjectsAsJSON(cmd.OutOrStdout(), result)
			case utils.YamlFormat:
				return utils.PrintObjectsAsYaml(cmd.OutOrStdout(), result)
			default:
				kvData := extractKVData(result)
				table := uitable.New()
				table.AddRow("KEY", "VALUE")
				for k, v := range kvData {
					table.AddRow(k, fmt.Sprintf("%v", v))
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

// extractKVData extracts the nested data.data map from a KV v2 response.
func extractKVData(result map[string]interface{}) map[string]interface{} {
	dataRaw, ok := result["data"]
	if !ok {
		return result
	}

	dataMap, ok := dataRaw.(map[string]interface{})
	if !ok {
		return result
	}

	innerData, ok := dataMap["data"]
	if !ok {
		return dataMap
	}

	innerMap, ok := innerData.(map[string]interface{})
	if !ok {
		return dataMap
	}

	return innerMap
}
