package actions

import (
	"context"
	"fmt"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/utils"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

const (
	integrationFilterFlagName = "integration"
	bundleFilterFlagName      = "bundle"
)

func AccessList() *cobra.Command {
	var integrationFilter string
	var bundleFilter string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all access sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			var integrationIDs []string
			if integrationFilter != "" {
				integrationIDs = []string{integrationFilter}
			}

			bundleIDsFilter := resolveBundleNameOrIDFlag(cmd.Context(), client, bundleFilter)

			accessSessions, err := services.ListAccessSessions(cmd.Context(), client, integrationIDs, bundleIDsFilter)
			if err != nil {
				return err
			}

			table := uitable.New()
			table.AddRow("ID", "NAME", "INTEGRATION NAME", "INTEGRATION TYPE", "TYPE")
			for _, session := range accessSessions {
				table.AddRow(session.Id, session.Name, session.Integration.Name, session.Integration.Type, session.Type.Name)
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
			if err != nil {
				return err
			}

			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&integrationFilter, integrationFilterFlagName, "i", "", "filter by integration name or id")
	flags.StringVarP(&bundleFilter, bundleFilterFlagName, "b", "", "filter by bundle name or id")

	return cmd
}

func resolveBundleNameOrIDFlag(ctx context.Context, client *aponoapi.AponoClient, bundleIDOrName string) []string {
	if bundleIDOrName == "" {
		return nil
	}

	if utils.IsValidUUID(bundleIDOrName) {
		return []string{bundleIDOrName}
	}

	bundles, err := services.ListBundles(ctx, client, bundleIDOrName)
	if err != nil {
		return []string{bundleIDOrName}
	}

	for _, bundle := range bundles {
		if bundle.Id == bundleIDOrName || bundle.Name == bundleIDOrName {
			return []string{bundle.Id}
		}
	}

	return []string{bundleIDOrName}
}
