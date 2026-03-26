package actions

import (
	"fmt"

	"github.com/spf13/cobra"
)

func ConfigGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long:  fmt.Sprintf("Get a configuration value.\n\nAvailable keys:\n%s", supportedKeysDescription()),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			handler, exists := configHandlers[key]
			if !exists {
				return fmt.Errorf("unknown configuration key: %s\n\nAvailable keys:\n%s", key, supportedKeysDescription())
			}

			value, err := handler.get()
			if err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", value)
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
