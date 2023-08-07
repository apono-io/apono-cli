package commands

import (
	"fmt"

	"github.com/apono-io/apono-sdk-go"
	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

func ListIntegrations() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "integrations",
		GroupID: Group.ID,
		Short:   "List all integrations available for requesting access",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			selectableIntegrationsResp, _, err := client.AccessRequestsApi.GetSelectableIntegrations(cmd.Context()).Execute()
			if err != nil {
				return err
			}

			resp, _, err := client.IntegrationsApi.ListIntegrationsV2(cmd.Context()).Execute()
			if err != nil {
				return err
			}

			integrations := make(map[string]apono.Integration)
			for _, val := range resp.Data {
				integrations[val.Id] = val
			}

			table := uitable.New()
			table.AddRow("ID", "TYPE", "NAME")
			for _, integrationID := range selectableIntegrationsResp.Data {
				if integration, ok := integrations[integrationID.Id]; ok {
					table.AddRow(integration.Id, integration.Type, integration.Name)
				}
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
