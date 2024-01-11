package interactive

import (
	"context"
	"fmt"
	listselect2 "github.com/apono-io/apono-cli/pkg/interactive/inputs/list_select"

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

	resourceTypeInput := listselect2.SelectInput[clientapi.ResourceTypeClientModel]{
		Title:       styles.BeforeSelectingItemsTitleStyle("Select resource type"),
		Options:     resourceTypes,
		FilterFunc:  func(s clientapi.ResourceTypeClientModel) string { return s.Name },
		DisplayFunc: func(s clientapi.ResourceTypeClientModel) string { return s.Name },
		IsEqual: func(s clientapi.ResourceTypeClientModel, s2 clientapi.ResourceTypeClientModel) bool {
			return s.Id == s2.Id
		},
		PostMessage: func(s []clientapi.ResourceTypeClientModel) string {
			return styles.AfterSelectingItemsTitleStyle("Selected resource type", []string{s[0].Name})
		},
		ShowHelp:      true,
		EnableFilter:  true,
		ShowItemCount: true,
	}

	selectedResourceTypes, err := listselect2.LaunchSelector(resourceTypeInput)
	if err != nil {
		return nil, err
	}

	return &selectedResourceTypes[0], nil
}
