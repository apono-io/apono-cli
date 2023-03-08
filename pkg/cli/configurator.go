package cli

import "github.com/spf13/cobra"

type Configurator interface {
	ConfigureCommands(rootCmd *cobra.Command) error
}
