package actions

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/config"
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

			// Validate that the token has the required scope for assist
			session, _ := config.GetCurrentProfile(cmd.Context())
			if !config.SessionHasScope(session, config.ScopeAssistant) {
				return fmt.Errorf("your current session doesn't have permission to use the assist feature\n\nPlease re-login to get the required permissions:\n  apono login")
			}

			return assist.RunAssistant(cmd.Context(), client)
		},
	}

	return cmd
}
