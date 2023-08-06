package commands

import (
	"fmt"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

func ListResources() *cobra.Command {
	var integrationID string
	var resourceType string
	cmd := &cobra.Command{
		Use:     "resources",
		GroupID: Group.ID,
		Short:   "List all resources of integration resource type",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			resp, _, err := client.AccessRequestsApi.GetSelectableResources(cmd.Context(), integrationID, resourceType).Execute()
			if err != nil {
				return err
			}

			table := uitable.New()
			table.AddRow("ID", "TYPE", "NAME")
			for _, resource := range resp.Data {
				table.AddRow(resource.Id, resource.Type, resource.Name)
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
			return err
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&integrationID, "integration", "i", "", "integration id")
	flags.StringVarP(&resourceType, "type", "t", "", "resource type")
	_ = cmd.MarkFlagRequired("integration")
	_ = cmd.MarkFlagRequired("type")

	return cmd
}
