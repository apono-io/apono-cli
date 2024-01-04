package actions

import (
	"github.com/spf13/cobra"
)

func Profiles() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "profiles",
		Short:   "Manage the CLI's profiles",
		Aliases: []string{"profile"},
		GroupID: Group.ID,
	}

	return cmd
}
