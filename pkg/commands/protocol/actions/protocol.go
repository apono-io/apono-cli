package actions

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/groups"
)

func Protocol() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "protocol",
		Short:   "Manage apono:// URI protocol handler (macOS)",
		GroupID: groups.OtherCommandsGroup.ID,
	}

	return cmd
}
