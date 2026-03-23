package services

import (
	"fmt"

	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/config"
	"github.com/apono-io/apono-cli/pkg/styles"
	"github.com/spf13/cobra"
)

const (
	LocationPostLogin   = "post_login"
	LocationPostRequest = "post_request"

	CategoryFeatureAnnouncement = "feature_announcement"
)

func FetchAndPrintNotifications(cmd *cobra.Command, client *clientapi.APIClient, location string) {
	resp, _, err := client.DefaultAPI.List(cmd.Context()).Location(location).Execute()
	if err != nil {
		return
	}

	var relevantNotifications []clientapi.CliNotificationClientModel
	for _, notification := range resp.Notifications {
		if isNotificationCategoryEnabled(notification.GetCategory()) {
			relevantNotifications = append(relevantNotifications, notification)
		}
	}

	for _, notification := range relevantNotifications {
		styledPrefix := styles.GetCustomMessagePrefix(notification.GetPrefix(), notification.GetPrefixColor())
		fmt.Fprintf(cmd.OutOrStdout(), "\n%s %s\n", styledPrefix, notification.GetText())
	}
}

func isNotificationCategoryEnabled(category string) bool {
	switch category {
	case CategoryFeatureAnnouncement:
		return config.IsFeatureAnnouncementNotificationsEnabled()
	default:
		return true
	}
}
