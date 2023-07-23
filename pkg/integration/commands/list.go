package commands

import (
	"context"
	"fmt"
	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
)

func List() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all integrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			return showIntegrationsSummary(cmd, client)
		},
	}

	return cmd
}

func showIntegrationsSummary(cmd *cobra.Command, client *aponoapi.AponoClient) error {
	integrations, err := listIntegrations(cmd.Context(), client)
	if err != nil {
		return err
	}

	selectableIntegrationsIds, err := listSelectableIntegrations(cmd.Context(), client)
	if err != nil {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "failed to fetch selectable integrations:", err)
		return err
	}

	integrationById := make(map[string]aponoapi.Integration)
	for _, integration := range integrations {
		integrationById[integration.Id] = integration
	}

	table := uitable.New()
	table.AddRow("ID", "NAME", "TYPE")
	for _, selectableIntegration := range selectableIntegrationsIds {
		integration, found := integrationById[selectableIntegration.Id]
		if !found {
			continue
		}

		table.AddRow(integration.Id, integration.Name, integration.Type)
	}

	_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
	return err
}

func listIntegrations(ctx context.Context, client *aponoapi.AponoClient) ([]aponoapi.Integration, error) {
	resp, err := client.ListIntegrationsV2WithResponse(ctx)
	if err != nil {
		return nil, err
	}

	return resp.JSON200.Data, nil
}
