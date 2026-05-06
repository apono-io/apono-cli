package actions

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/groups"
)

func AccessHandler() *cobra.Command {
	return &cobra.Command{
		Use:     "access-handler",
		Short:   "Manage the apono:// URL handler (macOS)",
		Hidden:  true,
		GroupID: groups.OtherCommandsGroup.ID,
	}
}
