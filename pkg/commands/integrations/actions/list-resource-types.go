package actions

import (
	"fmt"
	"github.com/apono-io/apono-cli/pkg/services"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

func ListResourceTypes() *cobra.Command {
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

			table := uitable.New()
			table.AddRow("ID", "NAME")
			for _, resourceType := range resourceTypes {
				table.AddRow(resourceType.Id, resourceType.Name)
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
			return err
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&integrationIDOrName, "integration", "i", "", "the integration id or type/name, for example: \"aws-account/My AWS integration\"")
	_ = cmd.MarkFlagRequired("integration")

	return cmd
}
