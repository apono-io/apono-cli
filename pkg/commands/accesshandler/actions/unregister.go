package actions

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/urihandler"
)

func Unregister() *cobra.Command {
	return &cobra.Command{
		Use:   "unregister",
		Short: "Remove the apono:// URL handler",
		RunE: func(cmd *cobra.Command, args []string) error {
			return urihandler.Unregister(cmd.OutOrStdout())
		},
	}
}
