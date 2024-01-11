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

//nolint:dupl // Remove duplication error
func RunResourcesSelector(ctx context.Context, client *aponoapi.AponoClient, integrationID string, resourceTypeID string) ([]clientapi.ResourceClientModel, error) {
	resources, err := services.ListResources(ctx, client, integrationID, resourceTypeID)
	if err != nil {
		return nil, err
	}
	if len(resources) == 0 {
		return nil, fmt.Errorf("no resources found for integration %s and resource type %s", integrationID, resourceTypeID)
	}

	resourceById := make(map[string]clientapi.ResourceClientModel)
	var options []listselect.SelectOption
	for _, resource := range resources {
		options = append(options, listselect.SelectOption{
			ID:     resource.Id,
			Label:  resource.Path,
			Filter: resource.Path,
		})
		resourceById[resource.Id] = resource
	}

	resourceInput := listselect.SelectInput{
		Title:             styles.BeforeSelectingItemsTitleStyle("Select resources"),
		Options:           options,
		MultipleSelection: true,
		PostMessage: func(s []listselect.SelectOption) string {
			var names []string
			for _, resource := range s {
				names = append(names, resource.Label)
			}
			return styles.AfterSelectingItemsTitleStyle("Selected resources", names)
		},
		ShowHelp:      true,
		EnableFilter:  true,
		ShowItemCount: true,
	}

	selectedItems, err := listselect.LaunchSelector(resourceInput)
	if err != nil {
		return nil, err
	}

	var selectedResources []clientapi.ResourceClientModel
	for _, selectedItem := range selectedItems {
		selectedResource, ok := resourceById[selectedItem.ID]
		if !ok {
			return nil, err
		}
		selectedResources = append(selectedResources, selectedResource)
	}

	return selectedResources, nil
}
