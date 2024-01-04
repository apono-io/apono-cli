package auth

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/groups"

	"github.com/apono-io/apono-cli/pkg/commands/auth/actions"
)

type Configurator struct{}

func (c *Configurator) ConfigureCommands(rootCmd *cobra.Command) error {
	rootCmd.AddGroup(groups.AuthCommandsGroup)
	profilesCommand := actions.Profiles()

	rootCmd.AddCommand(actions.Login())
	rootCmd.AddCommand(actions.Logout())
	rootCmd.AddCommand(profilesCommand)

	profilesCommand.AddCommand(actions.GetProfiles())
	profilesCommand.AddCommand(actions.SetProfile())
	return nil
}
