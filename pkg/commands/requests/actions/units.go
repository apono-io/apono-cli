package actions

import (
	"fmt"

	"github.com/apono-io/apono-cli/pkg/services"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

func AccessUnits() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "units <request_id>",
		Short:   "Return the access units details for the specified access request",
		Aliases: []string{"acccess-units", "access-unit", "accessunit", "accessunits"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("missing request ID")
			}

			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			requestID := args[0]
			return showRequestAccessUnits(cmd, client, requestID)
		},
	}

	return cmd
}

func showRequestAccessUnits(cmd *cobra.Command, client *aponoapi.AponoClient, requestID string) error {
	requestAccessUnits, err := services.ListAccessRequestAccessUnits(cmd.Context(), client, requestID)
	if err != nil {
		return err
	}

	table := uitable.New()
	table.AddRow("RESOURCE ID", "RESOURCE TYPE", "RESOURCE NAME", "PERMISSION", "INTEGRATION NAME", "INTEGRATION TYPE")
	for _, requestAccessUnit := range requestAccessUnits {
		table.AddRow(
			requestAccessUnit.Resource.Id,
			requestAccessUnit.Resource.Type.Id,
			requestAccessUnit.Resource.Name,
			requestAccessUnit.Permission.Id,
			requestAccessUnit.Resource.Integration.Name,
			requestAccessUnit.Resource.Integration.Type,
		)
	}

	_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
	return err
}
