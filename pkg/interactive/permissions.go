package interactive

import (
	"context"
	listselect2 "github.com/apono-io/apono-cli/pkg/interactive/inputs/list_select"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/styles"
)

//nolint:dupl // Remove duplication error
func RunPermissionsSelector(ctx context.Context, client *aponoapi.AponoClient, integrationID string, resourceTypeID string, multipleChoice bool) ([]clientapi.PermissionClientModel, error) {
	permissions, err := services.ListPermissions(ctx, client, integrationID, resourceTypeID)
	if err != nil {
		return nil, err
	}

	permissionsInput := listselect2.SelectInput[clientapi.PermissionClientModel]{
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

	selectedPermissions, err := listselect2.LaunchSelector(permissionsInput)
	if err != nil {
		return nil, err
	}

	return selectedPermissions, nil
}
