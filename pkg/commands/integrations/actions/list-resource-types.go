package actions

import (
	"fmt"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

func ListResourceTypes() *cobra.Command {
	var integrationID string
	cmd := &cobra.Command{
		Use:     "resource-types",
		GroupID: Group.ID,
		Short:   "ListIntegrations all resource types of integration",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			resp, _, err := client.AccessRequestsApi.GetSelectableResourceTypes(cmd.Context(), integrationID).Execute()
			if err != nil {
				return err
			}

			table := uitable.New()
			table.AddRow("ID", "NAME")
			for _, resourceType := range resp.Data {
				table.AddRow(resourceType.Id, resourceType.Name)
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
			return err
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&integrationID, "integration", "i", "", "integration id")
	_ = cmd.MarkFlagRequired("integration")

	return cmd
}
