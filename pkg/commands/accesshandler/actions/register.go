package actions

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/urihandler"
)

func Register() *cobra.Command {
	return &cobra.Command{
		Use:   "register",
		Short: "Register the apono:// URL handler with macOS",
		RunE: func(cmd *cobra.Command, args []string) error {
			return urihandler.Register(cmd.OutOrStdout())
		},
	}
}
