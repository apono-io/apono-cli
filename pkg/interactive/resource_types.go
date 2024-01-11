package interactive

import (
	"context"
	"fmt"
	listselect "github.com/apono-io/apono-cli/pkg/interactive/inputs/list_select"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/styles"
)

func RunResourceTypeSelector(ctx context.Context, client *aponoapi.AponoClient, integrationID string) (*clientapi.ResourceTypeClientModel, error) {
	resourceTypes, err := services.ListResourceTypes(ctx, client, integrationID)
	if err != nil {
		return nil, err
	}
	if len(resourceTypes) == 0 {
		return nil, fmt.Errorf("no resource types found for integration %s", integrationID)
	}

	resourceTypeById := make(map[string]clientapi.ResourceTypeClientModel)
	var options []listselect.SelectOption
	for _, resourceType := range resourceTypes {
		options = append(options, listselect.SelectOption{
			ID:     resourceType.Id,
			Label:  resourceType.Name,
			Filter: resourceType.Name,
		})
		resourceTypeById[resourceType.Id] = resourceType
	}

	resourceTypeInput := listselect.SelectInput{
		Title:   styles.BeforeSelectingItemsTitleStyle("Select resource type"),
		Options: options,
		PostMessage: func(s []listselect.SelectOption) string {
			return styles.AfterSelectingItemsTitleStyle("Selected resource type", []string{s[0].Label})
		},
		ShowHelp:      true,
		EnableFilter:  true,
		ShowItemCount: true,
	}

	selectedItems, err := listselect.LaunchSelector(resourceTypeInput)
	if err != nil {
		return nil, err
	}

	selectedResourceType, ok := resourceTypeById[selectedItems[0].ID]
	if !ok {
		return nil, fmt.Errorf("resource type not found")
	}

	return &selectedResourceType, nil
}
