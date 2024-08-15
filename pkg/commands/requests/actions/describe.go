package actions

import (
	"fmt"

	"github.com/apono-io/apono-cli/pkg/utils"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/services"

	"github.com/apono-io/apono-cli/pkg/clientapi"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

func Describe() *cobra.Command {
	format := new(utils.Format)

	cmd := &cobra.Command{
		Use:     "describe <request_id>",
		Short:   "Return the details for the specified access request",
		Aliases: []string{"get"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("missing request ID")
			}

			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			requestID := args[0]
			resp, _, err := client.ClientAPI.AccessRequestsAPI.GetAccessRequest(cmd.Context(), requestID).Execute()
			if err != nil {
				return err
			}

			err = services.PrintAccessRequests(cmd, []clientapi.AccessRequestClientModel{*resp}, *format, false)
			if err != nil {
				return err
			}

			if services.IsRequestWaitingForMFA(resp) && *format == utils.TableFormat {
				err = services.PrintAccessRequestMFALink(cmd, &resp.Id)
				if err != nil {
					return err
				}
			}

			return nil
		},
	}
	flags := cmd.Flags()
	utils.AddFormatFlag(flags, format)

	return cmd
}
