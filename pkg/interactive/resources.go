package interactive

import (
	"context"
	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/inputs/list_select"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/styles"
)

func runResourceSelector(ctx context.Context, client *aponoapi.AponoClient, integrationId string, resourceTypeId string) ([]clientapi.ResourceClientModel, error) {
	resources, err := services.ListResources(ctx, client, integrationId, resourceTypeId)
	if err != nil {
		return nil, err
	}

	resourceInput := list_select.SelectInput[clientapi.ResourceClientModel]{
		Title:             styles.BeforeSelectingItemsTitleStyle("Select resources"),
		Options:           resources,
		MultipleSelection: true,
		FilterFunc:        func(s clientapi.ResourceClientModel) string { return s.Path },
		DisplayFunc:       func(s clientapi.ResourceClientModel) string { return s.Path },
		IsEqual: func(s clientapi.ResourceClientModel, s2 clientapi.ResourceClientModel) bool {
			return s.Id == s2.Id
		},
		PostMessage: func(s []clientapi.ResourceClientModel) string {
			var names []string
			for _, resource := range s {
				names = append(names, resource.Path)
			}
			return styles.AfterSelectingItemsTitleStyle("Selected resources", names)
		},
		ShowHelp:      true,
		EnableFilter:  true,
		ShowItemCount: true,
	}

	selectedResources, err := list_select.LaunchSelector(resourceInput)
	if err != nil {
		return nil, err
	}

	return selectedResources, nil
}
