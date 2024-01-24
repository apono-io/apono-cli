package actions

import (
	"context"
	"fmt"

	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/utils"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

const (
	integrationFilterFlagName = "integration"
	bundleFilterFlagName      = "bundle"
	requestIDFlagName         = "request"
)

func AccessList() *cobra.Command {
	format := new(utils.Format)
	var integrationFilter string
	var bundleFilter string
	var requestFilter string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all access sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			integrationIDs := resolveIntegrationNameOrIDFlag(cmd.Context(), client, integrationFilter)
			bundleIDsFilter := resolveBundleNameOrIDFlag(cmd.Context(), client, bundleFilter)
			requestIDsFilter := resolveRequestIDFlag(requestFilter)

			accessSessions, err := services.ListAccessSessions(cmd.Context(), client, integrationIDs, bundleIDsFilter, requestIDsFilter)
			if err != nil {
				return err
			}

			if len(accessSessions) == 0 {
				return fmt.Errorf("no active access found, create a new request by running this command: apono request create")
			}

			err = services.PrintAccessSessionDetails(cmd, accessSessions, format)
			if err != nil {
				return err
			}

			return nil
		},
	}

	flags := cmd.Flags()
	utils.AddFormatFlag(flags, format)
	flags.StringVarP(&integrationFilter, integrationFilterFlagName, "i", "", "The integration id or type/name, for example: \"aws-account/My AWS integration\"")
	flags.StringVarP(&bundleFilter, bundleFilterFlagName, "b", "", "filter by bundle name or id")
	flags.StringVarP(&requestFilter, requestIDFlagName, "r", "", "filter by request id")

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
		if bundle.Name == bundleIDOrName {
			return []string{bundle.Id}
		}
	}

	return []string{bundleIDOrName}
}

func resolveIntegrationNameOrIDFlag(ctx context.Context, client *aponoapi.AponoClient, integrationIDOrName string) []string {
	if integrationIDOrName == "" {
		return nil
	}

	integration, err := services.GetIntegrationByIDOrByTypeAndName(ctx, client, integrationIDOrName)
	if err != nil {
		return []string{integrationIDOrName}
	}

	return []string{integration.Id}
}

func resolveRequestIDFlag(requestID string) []string {
	if requestID == "" {
		return nil
	}

	return []string{requestID}
}
