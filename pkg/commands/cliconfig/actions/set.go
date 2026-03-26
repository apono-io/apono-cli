package actions

import (
	"fmt"

	"github.com/spf13/cobra"
)

func ConfigSet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long:  fmt.Sprintf("Set a configuration value.\n\nAvailable keys:\n%s", supportedKeysDescription()),
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]

			handler, exists := configHandlers[key]
			if !exists {
				return fmt.Errorf("unknown configuration key: %s\n\nAvailable keys:\n%s", key, supportedKeysDescription())
			}

			if err := handler.apply(value); err != nil {
				return err
			}

			_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s set to %s\n", key, value)
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
