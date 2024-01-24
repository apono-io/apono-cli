package actions

import (
	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/utils"

	"github.com/spf13/cobra"
)

func List() *cobra.Command {
	format := new(utils.Format)
	var daysOffset int64

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all access request",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			requests, err := services.ListRequests(cmd.Context(), client, daysOffset)
			if err != nil {
				return err
			}

			err = services.PrintAccessRequestDetails(cmd, requests, *format, true)
			if err != nil {
				return err
			}

			return nil
		},
	}

	flags := cmd.Flags()
	flags.Int64VarP(&daysOffset, "days", "d", 7, "number of days to list")
	utils.AddFormatFlag(flags, format)

	return cmd
}
