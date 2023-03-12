package commands

import (
	"fmt"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/config"
)

func GetProfiles() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get-profiles",
		GroupID: Group.ID,
		Short:   "Describe one or many profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}

			authConfig := cfg.Auth
			table := uitable.New()
			table.MaxColWidth = 50

			table.AddRow("CURRENT", "NAME", "ACCOUNT ID", "USER ID", "CREATED")
			if authConfig.Profiles != nil {
				for name, profile := range authConfig.Profiles {
					var currentMark string
					if authConfig.ActiveProfile == name {
						currentMark = "*"
					}

					table.AddRow(currentMark, name, profile.AccountID, profile.UserID, profile.CreatedAt)
				}
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
			return err
		},
	}

	return cmd
}
