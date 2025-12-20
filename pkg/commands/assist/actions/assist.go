package actions

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/groups"
	"github.com/apono-io/apono-cli/pkg/interactive/assist"
)

func Assist() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "assist",
		Short:   "Interactive AI assistant for access management",
		Long:    `Start an interactive AI assistant session to help you request access, find resources, and manage your permissions through natural language.`,
		GroupID: groups.OtherCommandsGroup.ID,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			return assist.RunAssistant(cmd.Context(), client)
		},
	}

	return cmd
}
