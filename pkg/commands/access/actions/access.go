package actions

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/groups"
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
