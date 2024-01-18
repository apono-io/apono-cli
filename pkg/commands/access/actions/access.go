package actions

import (
	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/groups"
	"github.com/apono-io/apono-cli/pkg/interactive/flows"

	"github.com/spf13/cobra"
)

func Access() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "access",
		Short:   "Manage access to resources",
		GroupID: groups.ManagementCommandsGroup.ID,
		Aliases: []string{"sessions", "session"},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			err = flows.RunUseSessionInteractiveFlow(cmd, client, "")

			return err
		},
	}

	return cmd
}
