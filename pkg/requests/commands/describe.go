package commands

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

func Describe() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "describe <request_id>",
		Short:   "Return the details for the specified access request",
		Aliases: []string{"get"},
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
			return showRequestStatus(cmd, client, requestID)
		},
	}

	return cmd
}
