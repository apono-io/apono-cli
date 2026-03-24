package actions

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/config"
	"github.com/apono-io/apono-cli/pkg/groups"
)

const (
	keyNotificationsFeatureAnnouncements = "notifications.feature_announcements"
)

type configHandler struct {
	description string
	apply       func(value string) error
}

var configHandlers = map[string]configHandler{
	keyNotificationsFeatureAnnouncements: {
		description: "true/false - enable or disable feature announcement messages",
		apply: func(value string) error {
			if value != "true" && value != "false" {
				return fmt.Errorf("invalid value for %s: %s (expected true or false)", keyNotificationsFeatureAnnouncements, value)
			}
			return config.SetFeatureAnnouncementsNotification(value == "true")
		},
	},
}

func supportedKeysDescription() string {
	var parts []string
	for key, handler := range configHandlers {
		parts = append(parts, fmt.Sprintf("  %s (%s)", key, handler.description))
	}
	return strings.Join(parts, "\n")
}

func Config() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "config",
		Short:             "Manage the CLI's configuration",
		Long:              fmt.Sprintf("Manage the CLI's configuration.\n\nAvailable keys:\n%s", supportedKeysDescription()),
		GroupID:           groups.OtherCommandsGroup.ID,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error { return nil },
	}

	return cmd
}

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
