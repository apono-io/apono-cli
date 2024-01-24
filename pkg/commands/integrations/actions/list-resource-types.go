package actions

import (
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/utils"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
)

func ListResourceTypes() *cobra.Command {
	format := new(utils.Format)
	var integrationIDOrName string
	cmd := &cobra.Command{
		Use:     "resource-types",
		Short:   "List all resource types of integration",
		Aliases: []string{"resource-type"},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			integration, err := services.GetIntegrationByIDOrByTypeAndName(cmd.Context(), client, integrationIDOrName)
			if err != nil {
				return fmt.Errorf("failed to get integration: %w", err)
			}

			resourceTypes, err := services.ListResourceTypes(cmd.Context(), client, integration.Id)
			if err != nil {
				return err
			}

			switch *format {
			case utils.TableFormat:
				table := uitable.New()
				table.AddRow("ID", "NAME")
				for _, resourceType := range resourceTypes {
					table.AddRow(resourceType.Id, resourceType.Name)
				}

				_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
				return err
			case utils.JSONFormat:
				return utils.PrintObjectsAsJson(cmd.OutOrStdout(), resourceTypes)
			case utils.YamlFormat:
				return utils.PrintObjectsAsYaml(cmd.OutOrStdout(), resourceTypes)
			default:
				return fmt.Errorf("unsupported output format")
			}
		},
	}

	flags := cmd.Flags()
	utils.AddFormatFlag(flags, format)
	flags.StringVarP(&integrationIDOrName, "integration", "i", "", "the integration id or type/name, for example: \"aws-account/My AWS integration\"")
	_ = cmd.MarkFlagRequired("integration")

	return cmd
}
