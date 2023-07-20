package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

func Details() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "access-details [request-id]",
		GroupID: Group.ID,
		Short:   "Display access details of access request",
		Args:    cobra.ExactArgs(1), // This will enforce that exactly 1 argument must be provided
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			requestID := args[0]
			return showAccessDetails(cmd, client, requestID)
		},
	}

	return cmd
}

func showAccessDetails(cmd *cobra.Command, client *aponoapi.AponoClient, requestID string) error {
	resp, err := client.GetAccessRequestDetailsWithResponse(cmd.Context(), requestID)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(cmd.OutOrStdout(), resp.JSON200.Details)
	return err
}
