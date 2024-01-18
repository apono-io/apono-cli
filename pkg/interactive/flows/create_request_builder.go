package flows

import (
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/interactive/selectors"
	"github.com/apono-io/apono-cli/pkg/services"

	"github.com/spf13/cobra"
)

func StartRequestBuilderInteractiveMode(cmd *cobra.Command, client *aponoapi.AponoClient) (*clientapi.CreateAccessRequestClientModel, error) {
	requestType, err := selectors.RunRequestTypeSelector()
	if err != nil {
		return nil, err
	}
	switch requestType {
	case selectors.BundleRequestType:
		return StartBundleRequestBuilderInteractiveMode(cmd, client, "", "")
	case selectors.IntegrationRequestType:
		return StartIntegrationRequestBuilderInteractiveMode(cmd, client, "", "", []string{}, []string{}, "")
	default:
		return nil, fmt.Errorf("invalid request type: %s", requestType)
	}
}

func StartBundleRequestBuilderInteractiveMode(cmd *cobra.Command, client *aponoapi.AponoClient, bundleID string, justification string) (*clientapi.CreateAccessRequestClientModel, error) {
	request := services.GetEmptyNewRequestAPIModel()

	if bundleID == "" {
		bundle, err := selectors.RunBundleSelector(cmd.Context(), client)
		if err != nil {
			return nil, err
		}

		bundleID = bundle.Id
	}
	request.FilterBundleIds = []string{bundleID}

	if justification == "" {
		newJustification, err := selectors.RunJustificationInput()
		if err != nil {
			return nil, err
		}

		justification = newJustification
	}
	request.Justification = justification

	return request, nil
}

func StartIntegrationRequestBuilderInteractiveMode(
	cmd *cobra.Command,
	client *aponoapi.AponoClient,
	integrationID string,
	resourceTypeID string,
	resourceIDs []string,
	permissionIDs []string,
	justification string,
) (*clientapi.CreateAccessRequestClientModel, error) {
	request := services.GetEmptyNewRequestAPIModel()

	if integrationID == "" {
		integration, err := selectors.RunIntegrationSelector(cmd.Context(), client)
		if err != nil {
			return nil, err
		}

		integrationID = integration.Id
	}
	request.FilterIntegrationIds = []string{integrationID}

	var allowMultiplePermissions bool
	if resourceTypeID == "" {
		resourceType, err := selectors.RunResourceTypeSelector(cmd.Context(), client, integrationID)
		if err != nil {
			return nil, err
		}

		resourceTypeID = resourceType.Id
		allowMultiplePermissions = resourceType.AllowMultiplePermissions
	} else {
		resourceType, err := services.GetResourceTypeByID(cmd.Context(), client, integrationID, resourceTypeID)
		if err != nil {
			return nil, err
		}

		allowMultiplePermissions = resourceType.AllowMultiplePermissions
	}
	request.FilterResourceTypeIds = []string{resourceTypeID}

	if len(resourceIDs) == 0 {
		resources, err := selectors.RunResourcesSelector(cmd.Context(), client, integrationID, resourceTypeID)
		if err != nil {
			return nil, err
		}

		for _, resource := range resources {
			resourceIDs = append(resourceIDs, resource.Id)
		}
	}
	request.FilterResourceIds = resourceIDs

	if len(permissionIDs) == 0 {
		permissions, err := selectors.RunPermissionsSelector(cmd.Context(), client, integrationID, resourceTypeID, allowMultiplePermissions)
		if err != nil {
			return nil, err
		}

		for _, permission := range permissions {
			permissionIDs = append(permissionIDs, permission.Id)
		}
	}
	request.FilterPermissionIds = permissionIDs

	if justification == "" {
		newJustification, err := selectors.RunJustificationInput()
		if err != nil {
			return nil, err
		}

		justification = newJustification
	}
	request.Justification = justification

	return request, nil
}
