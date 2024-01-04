package actions

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

const (
	newCredentialsStatus = "new"
	maxWaitTime          = 30 * time.Second
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

			_, err = fmt.Fprintln(cmd.OutOrStdout(), "credentials reset request has been submitted, waiting for new credentials...")
			if err != nil {
				return err
			}

			startTime := time.Now()
			for {
				session, _, err := client.ClientAPI.AccessSessionsAPI.GetAccessSession(cmd.Context(), sessionID).Execute()
				if err != nil {
					return fmt.Errorf("access session with id %s not found", sessionID)
				}

				if session.Credentials.IsSet() && session.Credentials.Get().Status == newCredentialsStatus {
					break
				}

				time.Sleep(1 * time.Second)

				if time.Now().After(startTime.Add(maxWaitTime)) {
					return fmt.Errorf("timeout while waiting for credentials to reset")
				}
			}

			fmt.Println("credentials reset finished successfully")

			return nil
		},
	}

	return cmd
}
