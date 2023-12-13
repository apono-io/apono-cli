package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func Access() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "access",
		GroupID: Group.ID,
		Short:   "List all access sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("you must specify a subcommand: use, list, reset")
			return nil
		},
	}

	return cmd
}
