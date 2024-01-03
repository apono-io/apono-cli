package actions

import (
	"github.com/spf13/cobra"
)

func Access() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "access",
		GroupID: Group.ID,
		Short:   "The access command retrieves information about access sessions.",
		Aliases: []string{"sessions", "session"},
	}

	return cmd
}
