package interactive

import (
	"context"
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	listselect "github.com/apono-io/apono-cli/pkg/inputs/list_select"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/styles"
)

func RunIntegrationSelector(ctx context.Context, client *aponoapi.AponoClient) (*clientapi.IntegrationClientModel, error) {
	integrations, err := services.ListIntegrations(ctx, client)
	if err != nil {
		return nil, err
	}

	integrationInput := listselect.SelectInput[clientapi.IntegrationClientModel]{
		Title:       styles.BeforeSelectingItemsTitleStyle("Select integration"),
		Options:     integrations,
		FilterFunc:  func(s clientapi.IntegrationClientModel) string { return fmt.Sprintf("%s %s", s.Type, s.Name) },
		DisplayFunc: func(s clientapi.IntegrationClientModel) string { return fmt.Sprintf("%s/%s", s.Type, s.Name) },
		IsEqual: func(s clientapi.IntegrationClientModel, s2 clientapi.IntegrationClientModel) bool {
			return s.Id == s2.Id
		},
		PostMessage: func(s []clientapi.IntegrationClientModel) string {
			return styles.AfterSelectingItemsTitleStyle("Selected integration", []string{s[0].Name})
		},
		ShowHelp:      true,
		EnableFilter:  true,
		ShowItemCount: true,
	}

	selectedIntegrations, err := listselect.LaunchSelector(integrationInput)
	if err != nil {
		return nil, err
	}

	return &selectedIntegrations[0], nil
}
