package actions

import (
	"fmt"
	"time"

	"github.com/apono-io/apono-cli/pkg/services"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

const (
	waitFlagName        = "wait"
	waitTimeoutFlagName = "wait-timeout"
	defaultWaitTimeout  = 30 * time.Second
)

func Revoke() *cobra.Command {
	var wait bool
	var waitTimeout time.Duration

	cmd := &cobra.Command{
		Use:   "revoke <request_id>",
		Short: "Revoke the specified access request",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("missing request ID")
			}

			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			requestID := args[0]
			err = services.RevokeRequest(cmd.Context(), client, requestID)
			if err != nil {
				return err
			}

			if wait {
				err = waitForRequestToBeRevoked(cmd, client, requestID, waitTimeout)
				if err != nil {
					return err
				}

				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Request %s successfully revoked\n", requestID)
				if err != nil {
					return err
				}
			} else {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Request %s started revoking\n", requestID)
				if err != nil {
					return err
				}
			}

			return nil
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&wait, waitFlagName, "w", false, "wait for the request to be revoked")
	flags.DurationVarP(&waitTimeout, waitTimeoutFlagName, "t", defaultWaitTimeout, "timeout for waiting for the request to be revoked")

	return cmd
}

func waitForRequestToBeRevoked(cmd *cobra.Command, client *aponoapi.AponoClient, requestID string, timeout time.Duration) error {
	startTime := time.Now()
	for {
		accessRequest, _, err := client.ClientAPI.AccessRequestsAPI.GetAccessRequest(cmd.Context(), requestID).Execute()
		if err != nil {
			return err
		}

		if accessRequest.Status.Status == services.AccessRequestRevokedStatus {
			return nil
		}
		if accessRequest.Status.Status == services.AccessRequestFailedStatus {
			return fmt.Errorf("request failed to revoke")
		}

		time.Sleep(1 * time.Second)

		if time.Now().After(startTime.Add(timeout)) {
			return fmt.Errorf("timeout while waiting for request to be revoked")
		}
	}
}
