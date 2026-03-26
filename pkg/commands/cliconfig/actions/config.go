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

func Config() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "config",
		Short:             "Manage the CLI's configuration",
		GroupID:           groups.OtherCommandsGroup.ID,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error { return nil },
	}

	return cmd
}

type configHandler struct {
	description string
	apply       func(value string) error
	get         func() (string, error)
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
		get: func() (string, error) {
			return fmt.Sprintf("%t", config.IsFeatureAnnouncementNotificationsEnabled()), nil
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
