package actions

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/groups"
)

func Inventory() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "inventory",
		Short:   "Manage requestable objects such as resources, permissions, and integrations",
		GroupID: groups.ManagementCommandsGroup.ID,
	}

	return cmd
}
