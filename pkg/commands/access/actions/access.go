package actions

import (
	"github.com/apono-io/apono-cli/pkg/groups"
	"github.com/spf13/cobra"
)

func Access() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "access",
		Short:   "Manage access to resources",
		GroupID: groups.ManagementCommandsGroup.ID,
		Aliases: []string{"sessions", "session"},
	}

	return cmd
}
