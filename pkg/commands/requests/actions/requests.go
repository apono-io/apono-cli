package actions

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/groups"
)

func Requests() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "requests",
		Short:   "Manage your access requests",
		GroupID: groups.ManagementCommandsGroup.ID,
		Aliases: []string{"request"},
	}

	return cmd
}
