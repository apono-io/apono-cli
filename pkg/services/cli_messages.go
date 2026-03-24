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
	resp, _, err := client.DefaultAPI.ListCliNotifications(cmd.Context()).Execute()
	if err != nil {
		return
	}

	for _, notification := range resp.Notifications {
		categoryConfig, isSupported := supportedNotificationCategories[notification.GetCategory()]
		if !isSupported {
			continue
		}

		if !categoryConfig.isEnabled() {
			continue
		}

		styledPrefix := color.Bold.Sprintf("[") + categoryConfig.color.Sprint(categoryConfig.prefix) + color.Bold.Sprintf("]")

		_, err = fmt.Fprintf(cmd.OutOrStdout(), "\n%s %s\n", styledPrefix, notification.GetText())
		if err != nil {
			return
		}
	}

	// add another row spacing between the last notification print and later prints
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "\n")
	if err != nil {
		return
	}
}
