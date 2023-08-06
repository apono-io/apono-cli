package commands

import (
	"fmt"

	"github.com/apono-io/apono-sdk-go"
	"github.com/gookit/color"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

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
			table.AddRow("INTEGRATION ID", "TYPE", "NAME", "STATUS")
			for _, integrationID := range selectableIntegrationsResp.Data {
				if integration, ok := integrations[integrationID.Id]; ok {
					table.AddRow(integration.Id, integration.Type, integration.Name, coloredStatus(integration.Status))
				}
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
			return err
		},
	}

	return cmd
}

func coloredStatus(status apono.IntegrationStatus) string {
	statusTitle := cases.Title(language.English).String(string(status))
	switch status {
	case apono.INTEGRATIONSTATUS_ACTIVE:
		return color.Green.Sprint(statusTitle)
	case apono.INTEGRATIONSTATUS_ERROR:
		return color.Red.Sprint(statusTitle)
	case apono.INTEGRATIONSTATUS_WARNING, apono.INTEGRATIONSTATUS_REFRESHING:
		return color.Yellow.Sprint(statusTitle)
	case apono.INTEGRATIONSTATUS_INITIALIZING, apono.INTEGRATIONSTATUS_DISABLED:
		return color.Gray.Sprint(statusTitle)
	default:
		return statusTitle
	}
}
