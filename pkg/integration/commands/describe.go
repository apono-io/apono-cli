package commands

import (
	"fmt"
	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
	"strings"
)

func Describe() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "describe <integration id>",
		Aliases: []string{"get"},
		Short:   "Return the details for the specified integration",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			requestId := args[0]

			return showIntegrationSummary(cmd, client, requestId)
		},
	}

	return cmd
}

func showIntegrationSummary(cmd *cobra.Command, client *aponoapi.AponoClient, integrationId string) error {
	integration, err := getIntegration(cmd.Context(), client, integrationId)
	if err != nil {
		return err
	}

	selectablePermissions, err := listSelectablePermissions(cmd.Context(), client, integrationId)
	if err != nil {
		return err
	}

	selectableResource, err := listSelectableResources(cmd.Context(), client, integrationId)
	if err != nil {
		return err
	}

	var resourcesIds []string
	for _, resource := range selectableResource {
		resourcesIds = append(resourcesIds, resource.Id)
	}

	table := uitable.New()
	table.MaxColWidth = 100
	table.Wrap = true
	table.AddRow("ID:", integration.Id)
	table.AddRow("Name:", integration.Name)
	table.AddRow("Type:", integration.Type)
	table.AddRow("Permissions:", strings.Join(selectablePermissions, ", "))
	table.AddRow("Resources:", strings.Join(resourcesIds, ", "))

	_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
	return err
}
