package apono

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/apono-io/apono-cli/pkg/version"

	"github.com/apono-io/apono-cli/pkg/groups"

	"github.com/apono-io/apono-cli/pkg/utils"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func VersionCommand(info version.VersionInfo) *cobra.Command {
	format := new(utils.Format)
	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Print the version information",
		GroupID: groups.OtherCommandsGroup.ID,
		RunE: func(cmd *cobra.Command, args []string) error {
			switch *format {
			case utils.Plain:
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "Version: %s\n", info.Version)
				return err
			case utils.JSONFormat:
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				return encoder.Encode(info)
			case utils.YamlFormat:
				bytes, err := yaml.Marshal(info)
				if err != nil {
					return err
				}
				_, err = fmt.Fprint(cmd.OutOrStdout(), string(bytes))
				return err
			default:
				return errors.New("unsupported output format")
			}
		},
	}

	flags := cmd.PersistentFlags()
	utils.AddFormatFlag(flags, format)

	return cmd
}
