package actions

import (
	"fmt"

	"github.com/apono-io/apono-cli/pkg/services"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

func ListBundles() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bundles",
		Short:   "List all bundles available for requesting access",
		Aliases: []string{"bundle"},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			bundles, err := services.ListBundles(cmd.Context(), client)
			if err != nil {
				return err
			}

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
		},
	}

	return cmd
}
