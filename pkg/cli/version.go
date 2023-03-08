package cli

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/apono-io/apono-cli/pkg/output"
)

type VersionInfo struct {
	BuildDate string `json:"buildDate" yaml:"buildDate"`
	Commit    string `json:"commit" yaml:"commit"`
	Version   string `json:"version" yaml:"version"`
}

func VersionCommand(info VersionInfo) *cobra.Command {
	format := new(output.Format)
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			switch *format {
			case output.Plain:
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "Version: v%s\n", info.Version)
				return err
			case output.JSONFormat:
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				return encoder.Encode(info)
			case output.YamlFormat:
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
	output.AddFormatFlag(flags, format)

	return cmd
}
