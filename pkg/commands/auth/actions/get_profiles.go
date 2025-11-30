package actions

import (
	"fmt"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/config"
)

func GetProfiles() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "list",
		Short:             "List all profiles",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error { return nil },
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}

			authConfig := cfg.Auth
			table := uitable.New()
			table.MaxColWidth = 50

			table.AddRow("CURRENT", "NAME", "ACCOUNT", "USER", "CREATED")
			if authConfig.Profiles != nil {
				for name, profile := range authConfig.Profiles {
					var currentMark string
					if authConfig.ActiveProfile == name {
						currentMark = "*"
					}

					// Show names if available, fall back to IDs
					accountDisplay := profile.AccountName
					if accountDisplay == "" {
						accountDisplay = profile.AccountID
					}

					userDisplay := profile.UserEmail
					if userDisplay == "" {
						userDisplay = profile.UserID
					}

					table.AddRow(currentMark, name, accountDisplay, userDisplay, profile.CreatedAt)
				}
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
			return err
		},
	}

	return cmd
}
