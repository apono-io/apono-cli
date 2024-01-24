package actions

import (
	"fmt"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/utils"
)

func ListResources() *cobra.Command {
	format := new(utils.Format)
	var integrationIDOrName string
	var resourceType string
	cmd := &cobra.Command{
		Use:     "resources",
		Short:   "List all resources of integration resource type",
		Aliases: []string{"resource"},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			integration, err := services.GetIntegrationByIDOrByTypeAndName(cmd.Context(), client, integrationIDOrName)
			if err != nil {
				return fmt.Errorf("failed to get integration: %w", err)
			}

			resources, err := services.ListResources(cmd.Context(), client, integration.Id, resourceType, nil)
			if err != nil {
				return err
			}

			switch *format {
			case utils.TableFormat:
				table := uitable.New()
				table.AddRow("ID", "NAME")
				for _, resource := range resources {
					table.AddRow(resource.SourceId, resource.Name)
				}

				_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
				return err
			case utils.JSONFormat:
				return utils.PrintObjectsAsJson(cmd.OutOrStdout(), resources)
			case utils.YamlFormat:
				return utils.PrintObjectsAsYaml(cmd.OutOrStdout(), resources)
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
