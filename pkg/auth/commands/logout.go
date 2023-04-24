package commands

import (
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/config"
)

func Logout() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "logout [NAME]",
		GroupID:           Group.ID,
		Short:             "Logout from Apono",
		Args:              cobra.MaximumNArgs(1),
		PersistentPreRunE: func(_ *cobra.Command, args []string) error { return nil },
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}

			authConfig := &cfg.Auth
			profileName := authConfig.ActiveProfile
			if len(args) > 0 {
				profileName = config.ProfileName(args[0])
			}

			if authConfig.Profiles == nil {
				return aponoapi.ErrProfileNotExists
			}

			if _, exists := authConfig.Profiles[profileName]; !exists {
				return aponoapi.ErrProfileNotExists
			}

			delete(authConfig.Profiles, profileName)

			_, err = fmt.Fprintln(cmd.OutOrStdout(), "Logging out profile:", profileName)
			if err != nil {
				return err
			}

			return config.Save(cfg)
		},
	}

	return cmd
}
