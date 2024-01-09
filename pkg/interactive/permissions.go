package interactive

import (
	"context"
	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/inputs/list_select"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/styles"
)

func RunPermissionSelector(ctx context.Context, client *aponoapi.AponoClient, integrationId string, resourceTypeId string, multipleChoice bool) ([]clientapi.PermissionClientModel, error) {
	permissions, err := services.ListPermissions(ctx, client, integrationId, resourceTypeId)
	if err != nil {
		return nil, err
	}

	permissionsInput := list_select.SelectInput[clientapi.PermissionClientModel]{
		Title:             styles.BeforeSelectingItemsTitleStyle("Select permissions"),
		Options:           permissions,
		MultipleSelection: multipleChoice,
		FilterFunc:        func(s clientapi.PermissionClientModel) string { return s.Name },
		DisplayFunc:       func(s clientapi.PermissionClientModel) string { return s.Name },
		IsEqual: func(s clientapi.PermissionClientModel, s2 clientapi.PermissionClientModel) bool {
			return s.Id == s2.Id
		},
		PostMessage: func(s []clientapi.PermissionClientModel) string {
			var names []string
			for _, permission := range s {
				names = append(names, permission.Name)
			}
			return styles.AfterSelectingItemsTitleStyle("Selected permissions", names)
		},
		ShowHelp:      true,
		EnableFilter:  true,
		ShowItemCount: true,
	}

	selectedPermissions, err := list_select.LaunchSelector(permissionsInput)
	if err != nil {
		return nil, err
	}

	return selectedPermissions, nil
}
