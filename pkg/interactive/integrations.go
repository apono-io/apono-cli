package interactive

import (
	"context"
	"fmt"

	listselect "github.com/apono-io/apono-cli/pkg/interactive/inputs/list_select"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/services"
)

func RunIntegrationSelector(ctx context.Context, client *aponoapi.AponoClient) (*clientapi.IntegrationClientModel, error) {
	integrations, err := services.ListIntegrations(ctx, client)
	if err != nil {
		return nil, err
	}
	if len(integrations) == 0 {
		return nil, fmt.Errorf("no integrations found")
	}

	integrationByID := make(map[string]clientapi.IntegrationClientModel)
	var options []listselect.SelectOption
	for _, integration := range integrations {
		options = append(options, listselect.SelectOption{
			ID:    integration.Id,
			Label: fmt.Sprintf("%s/%s", integration.Type, integration.Name),
		})
		integrationByID[integration.Id] = integration
	}

	integrationInput := listselect.SelectInput{
		Title:         "Select integration",
		PostTitle:     "Selected integration",
		Options:       options,
		ShowHelp:      true,
		EnableFilter:  true,
		ShowItemCount: true,
	}

	selectedItems, err := listselect.LaunchSelector(integrationInput)
	if err != nil {
		return nil, err
	}

	selectedIntegration, ok := integrationByID[selectedItems[0].ID]
	if !ok {
		return nil, fmt.Errorf("integration not found")
	}

	return &selectedIntegration, nil
}
