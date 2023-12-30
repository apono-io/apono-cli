package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/utils"

	"github.com/spf13/cobra"
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
				err := cmd.Help()
				if err != nil {
					return err
				}

				os.Exit(0)
			}

			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			requestID := args[0]
			err = utils.RevokeRequest(cmd.Context(), client, requestID)
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

		if accessRequest.Status.Status == clientapi.ACCESSSTATUS_EXPIRED {
			return nil
		}
		if accessRequest.Status.Status == clientapi.ACCESSSTATUS_FAILED {
			return fmt.Errorf("request failed to revoke")
		}

		time.Sleep(1 * time.Second)

		if time.Now().After(startTime.Add(timeout)) {
			return fmt.Errorf("timeout while waiting for request to be revoked")
		}
	}
}
