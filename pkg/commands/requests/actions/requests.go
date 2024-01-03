package actions

import (
	"github.com/spf13/cobra"
)

func Requests() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "requests",
		GroupID: Group.ID,
		Short:   "The requests command retrieves, creates and modifies access requests.",
		Aliases: []string{"request"},
	}

	return cmd
}
