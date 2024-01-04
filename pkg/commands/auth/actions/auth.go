package actions

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/groups"
)

func Profiles() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "profiles",
		Short:   "Manage the CLI's profiles",
		Aliases: []string{"profile"},
		GroupID: groups.AuthCommandsGroup.ID,
	}

	return cmd
}
