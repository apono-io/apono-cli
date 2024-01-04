package actions

import (
	"github.com/apono-io/apono-cli/pkg/groups"

	"github.com/spf13/cobra"
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
