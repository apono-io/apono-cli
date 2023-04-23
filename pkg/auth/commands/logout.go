package commands

import (
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/config"
)

func Logout() *cobra.Command {
	profileName := new(string)
	cmd := &cobra.Command{
		Use:     "logout",
		GroupID: Group.ID,
		Short:   "Logout from Apono",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}

			authConfig := &cfg.Auth
			pn := config.ProfileName(*profileName)
			if *profileName == "current-profile" {
				pn = authConfig.ActiveProfile
			}

			if authConfig.Profiles == nil {
				return aponoapi.ErrProfileNotExists
			}

			if _, exists := authConfig.Profiles[pn]; !exists {
				return aponoapi.ErrProfileNotExists
			}
			delete(authConfig.Profiles, pn)

			_, err = fmt.Fprintln(cmd.OutOrStdout(), "Logging out profile:", pn)
			if err != nil {
				return err
			}

			return config.Save(cfg)
		},
	}

	cmd.Flags().StringVarP(profileName, "profile", "p", "current-profile", "Profile name")
	return cmd
}
