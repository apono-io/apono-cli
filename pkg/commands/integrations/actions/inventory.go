package actions

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/groups"
)

func Inventory() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "inventory",
		Short:   "List resources you can request access to: bundles, integrations, resource types, resources, and permissions",
		GroupID: groups.ManagementCommandsGroup.ID,
	}

	return cmd
}
