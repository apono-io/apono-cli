package services

import (
	"fmt"

	"github.com/gookit/color"
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/config"
)

const (
	CategoryFeatureAnnouncement = "featureAnnouncement"
)

type notificationCategoryConfig struct {
	prefix    string
	color     color.Color
	isEnabled func() bool
}

var supportedNotificationCategories = map[string]notificationCategoryConfig{
	CategoryFeatureAnnouncement: {
		prefix:    "NEW",
		color:     color.Magenta,
		isEnabled: config.IsFeatureAnnouncementNotificationsEnabled,
	},
}

// FetchAndPrintNotifications fetch notifications from server and prints them to the user if they are supported and
// enabled. Errors during fetching or printing are silently ignored to avoid interrupting the main flow of the CLI.
func FetchAndPrintNotifications(cmd *cobra.Command, client *clientapi.APIClient) {
	resp, _, err := client.DefaultAPI.List(cmd.Context()).Execute()
	if err != nil {
		return
	}

	for i, notification := range resp.Notifications {
		categoryConfig, isSupported := supportedNotificationCategories[notification.GetCategory()]
		if !isSupported {
			continue
		}

		if !categoryConfig.isEnabled() {
			continue
		}

		styledPrefix := color.Bold.Sprintf("[") + categoryConfig.color.Sprint(categoryConfig.prefix) + color.Bold.Sprintf("]")

		newLines := "\n"
		isLastNotification := i == len(resp.Notifications)-1
		if isLastNotification {
			newLines += "\n"
		}

		_, err := fmt.Fprintf(cmd.OutOrStdout(), "\n%s %s%s", styledPrefix, notification.GetText(), newLines)
		if err != nil {
			return
		}
	}
}
