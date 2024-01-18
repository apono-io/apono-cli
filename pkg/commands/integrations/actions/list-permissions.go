package actions

import (
	"fmt"

	"github.com/apono-io/apono-cli/pkg/services"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

func ListPermissions() *cobra.Command {
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

			table := uitable.New()
			table.AddRow("ID", "NAME")
			for _, permission := range permissions {
				table.AddRow(permission.Id, permission.Name)
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
			return err
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&integrationIDOrName, "integration", "i", "", "the integration id or type/name, for example: \"aws-account/My AWS integration\"")
	flags.StringVarP(&resourceType, "type", "t", "", "the resource type")
	_ = cmd.MarkFlagRequired("integration")
	_ = cmd.MarkFlagRequired("type")

	return cmd
}
