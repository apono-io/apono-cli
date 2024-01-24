package actions

import (
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/utils"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
)

func ListPermissions() *cobra.Command {
	format := new(utils.Format)
	var integrationIDOrName string
	var resourceType string
	cmd := &cobra.Command{
		Use:     "permissions",
		Short:   "List all permissions of integration resource type",
		Aliases: []string{"permission"},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			integration, err := services.GetIntegrationByIDOrByTypeAndName(cmd.Context(), client, integrationIDOrName)
			if err != nil {
				return fmt.Errorf("failed to get integration: %w", err)
			}

			permissions, err := services.ListPermissions(cmd.Context(), client, integration.Id, resourceType)
			if err != nil {
				return err
			}

			switch *format {
			case utils.TableFormat:
				table := uitable.New()
				table.AddRow("ID", "NAME")
				for _, permission := range permissions {
					table.AddRow(permission.Id, permission.Name)
				}

				_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
				return err
			case utils.JSONFormat:
				return utils.PrintObjectsAsJson(cmd.OutOrStdout(), permissions)
			case utils.YamlFormat:
				return utils.PrintObjectsAsYaml(cmd.OutOrStdout(), permissions)
			default:
				return fmt.Errorf("unsupported output format")
			}
		},
	}

	flags := cmd.Flags()
	utils.AddFormatFlag(flags, format)
	flags.StringVarP(&integrationIDOrName, "integration", "i", "", "the integration id or type/name, for example: \"aws-account/My AWS integration\"")
	flags.StringVarP(&resourceType, "type", "t", "", "the resource type")
	_ = cmd.MarkFlagRequired("integration")
	_ = cmd.MarkFlagRequired("type")

	return cmd
}
