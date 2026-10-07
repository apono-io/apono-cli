package actions

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/services"
)

func AccessReset() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset-credentials [id]",
		Short: "Reset access session credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("missing session id")
			}

			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), "credentials reset request has been submitted, waiting for new credentials...")
			if err != nil {
				return err
			}

			err = services.ResetSessionCredentials(cmd.Context(), client, args[0])
			if err != nil {
				return err
			}

			fmt.Println("credentials reset finished successfully")

			return nil
		},
	}

	return cmd
}
