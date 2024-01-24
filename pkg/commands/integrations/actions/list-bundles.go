package actions

import (
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/utils"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
)

func ListBundles() *cobra.Command {
	format := new(utils.Format)
	cmd := &cobra.Command{
		Use:     "bundles",
		Short:   "List all bundles available for requesting access",
		Aliases: []string{"bundle"},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			bundles, err := services.ListBundles(cmd.Context(), client, "")
			if err != nil {
				return err
			}

			switch *format {
			case utils.TableFormat:
				table := uitable.New()
				table.AddRow("ID", "NAME")
				for _, bundle := range bundles {
					table.AddRow(bundle.Id, bundle.Name)
				}

				_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
				if err != nil {
					return err
				}

				return nil
			case utils.JSONFormat:
				return utils.PrintObjectsAsJSON(cmd.OutOrStdout(), bundles)
			case utils.YamlFormat:
				return utils.PrintObjectsAsYaml(cmd.OutOrStdout(), bundles)
			default:
				return fmt.Errorf("unsupported output format")
			}
		},
	}
	flags := cmd.Flags()
	utils.AddFormatFlag(flags, format)

	return cmd
}
