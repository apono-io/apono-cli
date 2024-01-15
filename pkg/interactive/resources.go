package interactive

import (
	"context"
	"fmt"

	listselect "github.com/apono-io/apono-cli/pkg/interactive/inputs/list_select"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/services"
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

	resourceByID := make(map[string]clientapi.ResourceClientModel)
	var options []listselect.SelectOption
	for _, resource := range resources {
		options = append(options, listselect.SelectOption{
			ID:    resource.Id,
			Label: resource.Path,
		})
		resourceByID[resource.Id] = resource
	}

	resourceInput := listselect.SelectInput{
		Title:             "Select resources",
		PostTitle:         "Selected resources",
		Options:           options,
		MultipleSelection: true,
		ShowHelp:          true,
		EnableFilter:      true,
		ShowItemCount:     true,
	}

	selectedItems, err := listselect.LaunchSelector(resourceInput)
	if err != nil {
		return nil, err
	}

	var selectedResources []clientapi.ResourceClientModel
	for _, selectedItem := range selectedItems {
		selectedResource, ok := resourceByID[selectedItem.ID]
		if !ok {
			return nil, err
		}
		selectedResources = append(selectedResources, selectedResource)
	}

	return selectedResources, nil
}
