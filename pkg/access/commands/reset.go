package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

func AccessReset() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reset-credentials [id]",
		GroupID: Group.ID,
		Short:   "Reset access session credentials",
		Args:    cobra.ExactArgs(1), // This will enforce that exactly 1 argument must be provided
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			id := args[0]
			_, _, err = client.ClientAPI.AccessSessionsAPI.ResetAccessSessionCredentials(cmd.Context(), id).Execute()
			if err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), "credentials reset request has been submitted")
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
