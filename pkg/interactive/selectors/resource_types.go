package selectors

import (
	"context"
	"fmt"

	listselect "github.com/apono-io/apono-cli/pkg/interactive/inputs/list_select"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/services"
)

func RunResourceTypeSelector(ctx context.Context, client *aponoapi.AponoClient, integrationID string) (*clientapi.ResourceTypeClientModel, error) {
	resourceTypes, err := services.ListResourceTypes(ctx, client, integrationID)
	if err != nil {
		return nil, err
	}
	if len(resourceTypes) == 0 {
		return nil, fmt.Errorf("no resource types found for integration %s", integrationID)
	}

	resourceTypeByID := make(map[string]clientapi.ResourceTypeClientModel)
	var options []listselect.SelectOption
	for _, resourceType := range resourceTypes {
		options = append(options, listselect.SelectOption{
			ID:    resourceType.Id,
			Label: resourceType.Name,
		})
		resourceTypeByID[resourceType.Id] = resourceType
	}

	resourceTypeInput := listselect.SelectInput{
		Title:                "Select resource type",
		PostTitle:            "Selected resource type",
		Options:              options,
		ShowHelp:             true,
		EnableFilter:         true,
		ShowItemCount:        true,
		AutoSelectSingleItem: true,
	}

	selectedItems, err := listselect.LaunchSelector(resourceTypeInput)
	if err != nil {
		return nil, err
	}

	selectedResourceType, ok := resourceTypeByID[selectedItems[0].ID]
	if !ok {
		return nil, fmt.Errorf("resource type not found")
	}

	return &selectedResourceType, nil
}
