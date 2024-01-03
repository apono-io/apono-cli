package actions

import (
	"github.com/apono-io/apono-cli/pkg/groups"
	"github.com/spf13/cobra"
)

func Requests() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "requests",
		Short:   "Manage your access requests.",
		GroupID: groups.ManagementCommandsGroup.ID,
		Aliases: []string{"request"},
	}

	return cmd
}
