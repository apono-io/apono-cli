package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

const (
	newCredentialsStatus = "new"
	maxWaitAttempts      = 30
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

			sessionID := args[0]

			_, _, err = client.ClientAPI.AccessSessionsAPI.ResetAccessSessionCredentials(cmd.Context(), sessionID).Execute()
			if err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), "credentials reset request has been submitted, waiting until finished...")
			if err != nil {
				return err
			}

			retries := 0
			for {
				session, _, err := client.ClientAPI.AccessSessionsAPI.GetAccessSession(cmd.Context(), sessionID).Execute()
				if err != nil {
					return fmt.Errorf("access session with id %s not found", sessionID)
				}

				if session.Credentials.IsSet() && session.Credentials.Get().Status == newCredentialsStatus {
					break
				}

				time.Sleep(1 * time.Second)

				retries++
				if retries > maxWaitAttempts {
					return fmt.Errorf("timeout while waiting for credentials to reset")
				}
			}

			fmt.Println("credentials reset finished successfully")

			return nil
		},
	}

	return cmd
}
