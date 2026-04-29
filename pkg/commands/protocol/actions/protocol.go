package actions

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/groups"
)

const darwinOS = "darwin"

func Protocol() *cobra.Command {
	return &cobra.Command{
		Use:     "protocol",
		Short:   "Manage the apono:// URL handler (macOS)",
		GroupID: groups.OtherCommandsGroup.ID,
	}
}
