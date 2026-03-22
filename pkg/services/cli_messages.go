package services

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/config"
	"github.com/apono-io/apono-cli/pkg/styles"
)

func FetchAndDisplayMessages(cmd *cobra.Command, client *clientapi.APIClient, ctx context.Context, location string) {
	if !config.IsFeatureAnnouncementsEnabled() {
		return
	}

	resp, _, err := client.DefaultAPI.GetMessages(ctx).Location(location).Execute()
	if err != nil {
		return
	}

	for _, msg := range resp.Messages {
		var prefix string
		if msg.HasPrefix() {
			p := msg.GetPrefix()
			prefix = p
		}
		styledPrefix := styles.GetCustomMessagePrefix(prefix)
		fmt.Fprintf(cmd.OutOrStdout(), "\n%s %s\n", styledPrefix, msg.GetText())
	}
}
