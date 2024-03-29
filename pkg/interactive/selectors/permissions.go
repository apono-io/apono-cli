package selectors

import (
	"context"
	"fmt"

	listselect "github.com/apono-io/apono-cli/pkg/interactive/inputs/list_select"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/services"
)

func RunPermissionsSelector(ctx context.Context, client *aponoapi.AponoClient, integrationID string, resourceTypeID string, multipleChoice bool) ([]clientapi.PermissionClientModel, error) {
	permissions, err := services.ListPermissions(ctx, client, integrationID, resourceTypeID)
	if err != nil {
		return nil, err
	}
	if len(permissions) == 0 {
		return nil, fmt.Errorf("no permissions found for integration %s and resource type %s", integrationID, resourceTypeID)
	}

	permissionByID := make(map[string]clientapi.PermissionClientModel)
	var options []listselect.SelectOption
	for _, permission := range permissions {
		options = append(options, listselect.SelectOption{
			ID:    permission.Id,
			Label: permission.Name,
		})
		permissionByID[permission.Id] = permission
	}

	permissionsInput := listselect.SelectInput{
		Title:                "Select permissions",
		PostTitle:            "Selected permissions",
		Options:              options,
		MultipleSelection:    multipleChoice,
		ShowHelp:             true,
		EnableFilter:         true,
		ShowItemCount:        true,
		AutoSelectSingleItem: true,
	}

	selectedItems, err := listselect.LaunchSelector(permissionsInput)
	if err != nil {
		return nil, err
	}

	var selectedPermissions []clientapi.PermissionClientModel
	for _, selectedItem := range selectedItems {
		selectedPermission, ok := permissionByID[selectedItem.ID]
		if !ok {
			return nil, err
		}
		selectedPermissions = append(selectedPermissions, selectedPermission)
	}

	return selectedPermissions, nil
}
