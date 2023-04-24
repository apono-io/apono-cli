package commands

import (
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/config"
)

func SetProfile() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "set-profile NAME",
		GroupID:           Group.ID,
		Short:             "Set active profile",
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error { return nil },
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}

			pn := config.ProfileName(args[0])
			authConfig := &cfg.Auth
			if authConfig.Profiles == nil {
				return aponoapi.ErrProfileNotExists
			}

			if _, exists := authConfig.Profiles[pn]; !exists {
				return aponoapi.ErrProfileNotExists
			}

			authConfig.ActiveProfile = pn
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "Setting active profile to:", pn)
			if err != nil {
				return err
			}

			return config.Save(cfg)
		},
	}

	return cmd
}
