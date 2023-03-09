package commands

import (
	"fmt"
	"github.com/apono-io/apono-cli/pkg/config"
	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
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

			table.AddRow("CURRENT", "NAME", "CREATED")
			if authConfig.Profiles != nil {
				for name, profile := range authConfig.Profiles {
					var currentMark string
					if authConfig.ActiveProfile == name {
						currentMark = "*"
					}

					table.AddRow(currentMark, name, profile.CreatedAt)
				}
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
			return err
		},
	}

	return cmd
}
