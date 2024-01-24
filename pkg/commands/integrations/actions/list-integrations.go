package actions

import (
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/utils"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
)

func ListIntegrations() *cobra.Command {
	format := new(utils.Format)
	cmd := &cobra.Command{
		Use:     "integrations",
		Short:   "List all integrations available for requesting access",
		Aliases: []string{"integration"},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			integrations, err := services.ListIntegrations(cmd.Context(), client)
			if err != nil {
				return err
			}

			switch *format {
			case utils.TableFormat:
				table := uitable.New()
				table.AddRow("ID", "TYPE", "NAME")
				for _, integration := range integrations {
					table.AddRow(integration.Id, integration.Type, integration.Name)
				}

				_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
				if err != nil {
					return err
				}

				return nil
			case utils.JSONFormat:
				return utils.PrintObjectsAsJSON(cmd.OutOrStdout(), integrations)
			case utils.YamlFormat:
				return utils.PrintObjectsAsYaml(cmd.OutOrStdout(), integrations)
			default:
				return fmt.Errorf("unsupported output format")
			}
		},
	}
	flags := cmd.Flags()
	utils.AddFormatFlag(flags, format)

	return cmd
}
