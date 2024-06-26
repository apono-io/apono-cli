package flows

import (
	"fmt"
	"strings"
	"time"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/interactive/selectors"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/styles"

	"github.com/gookit/color"
	"github.com/spf13/cobra"
)

type CreateAccessRequestWithFullModels struct {
	Bundles      []clientapi.BundleClientModel
	Integrations []clientapi.IntegrationClientModel
	Resources    []clientapi.ResourceClientModel
	Duration     *time.Duration
}

func StartRequestBuilderInteractiveMode(cmd *cobra.Command, client *aponoapi.AponoClient) (*clientapi.CreateAccessRequestClientModel, error) {
	requestType, err := selectors.RunRequestTypeSelector()
	if err != nil {
		return nil, err
	}

	var request *clientapi.CreateAccessRequestClientModel
	switch requestType {
	case selectors.BundleRequestType:
		request, err = StartBundleRequestBuilderInteractiveMode(cmd, client, "", "", nil)
		if err != nil {
			return nil, err
		}
	case selectors.IntegrationRequestType:
		request, err = StartIntegrationRequestBuilderInteractiveMode(cmd, client, "", "", []string{}, []string{}, "", nil)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid request type: %s", requestType)
	}

	return request, nil
}

func StartBundleRequestBuilderInteractiveMode(
	cmd *cobra.Command,
	client *aponoapi.AponoClient,
	bundleID string,
	justification string,
	accessDuration *time.Duration,
) (*clientapi.CreateAccessRequestClientModel, error) {
	request := services.GetEmptyNewRequestAPIModel()
	requestModels := &CreateAccessRequestWithFullModels{}

	if bundleID == "" {
		bundle, err := selectors.RunBundleSelector(cmd.Context(), client)
		if err != nil {
			return nil, err
		}

		bundleID = bundle.Id
		requestModels.Bundles = []clientapi.BundleClientModel{*bundle}
	}
	request.FilterBundleIds = []string{bundleID}

	var justificationOptional bool
	var durationRequired bool
	var maxRequestDuration time.Duration
	dryRunResp, err := services.DryRunRequest(cmd.Context(), client, request)
	if err == nil {
		justificationOptional = services.IsJustificationOptionalForRequest(dryRunResp)
		durationRequired = services.IsDurationRequiredForRequest(dryRunResp)
		maxRequestDuration = services.GetMaximumRequestDuration(dryRunResp)
	}

	if accessDuration == nil && durationRequired {
		var newDuration *time.Duration
		newDuration, err = selectors.RunDurationInput(!durationRequired, 0, maxRequestDuration.Hours())
		if err != nil {
			return nil, err
		}

		accessDuration = newDuration
	}
	if accessDuration != nil {
		requestModels.Duration = accessDuration
		accessDurationInSec := int32(accessDuration.Seconds())
		request.DurationInSec = *clientapi.NewNullableInt32(&accessDurationInSec)
	}

	if justification == "" {
		var newJustification string
		newJustification, err = selectors.RunJustificationInput(justificationOptional)
		if err != nil {
			return nil, err
		}

		justification = newJustification
	}
	request.Justification = *clientapi.NewNullableString(&justification)

	err = GenerateAndPrintCreateRequestCommand(cmd, request, requestModels)
	if err != nil {
		return nil, err
	}

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
	accessDuration *time.Duration,
) (*clientapi.CreateAccessRequestClientModel, error) {
	request := services.GetEmptyNewRequestAPIModel()
	requestModels := &CreateAccessRequestWithFullModels{}

	if integrationID == "" {
		integration, err := selectors.RunIntegrationSelector(cmd.Context(), client)
		if err != nil {
			return nil, err
		}

		integrationID = integration.Id
		requestModels.Integrations = []clientapi.IntegrationClientModel{*integration}
	}
	request.FilterIntegrationIds = []string{integrationID}

	var allowMultiplePermissions bool
	var resourceType *clientapi.ResourceTypeClientModel
	if resourceTypeID == "" {
		var err error
		resourceType, err = selectors.RunResourceTypeSelector(cmd.Context(), client, integrationID)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		resourceType, err = services.GetResourceTypeByID(cmd.Context(), client, integrationID, resourceTypeID)
		if err != nil {
			return nil, err
		}
	}
	allowMultiplePermissions = resourceType.AllowMultiplePermissions
	request.FilterResourceTypeIds = []string{resourceType.Id}

	var resolvedResourceIDs []string
	if len(resourceIDs) == 0 {
		resources, err := selectors.RunResourcesSelector(cmd.Context(), client, integrationID, resourceType.Id)
		if err != nil {
			return nil, err
		}

		for _, resource := range resources {
			resolvedResourceIDs = append(resolvedResourceIDs, resource.Id)
		}
		requestModels.Resources = resources
	} else {
		resources, err := services.ListResourcesBySourceIDs(cmd.Context(), client, integrationID, resourceType.Id, resourceIDs)
		if err != nil {
			return nil, err
		}

		for _, resource := range resources {
			resolvedResourceIDs = append(resolvedResourceIDs, resource.Id)
		}
		requestModels.Resources = resources
	}
	request.FilterResources = services.ListResourceFiltersFromResourcesIDs(resolvedResourceIDs)

	if len(permissionIDs) == 0 {
		permissions, err := selectors.RunPermissionsSelector(cmd.Context(), client, integrationID, resourceType.Id, allowMultiplePermissions)
		if err != nil {
			return nil, err
		}

		for _, permission := range permissions {
			permissionIDs = append(permissionIDs, permission.Id)
		}
	}
	request.FilterPermissionIds = permissionIDs

	var justificationOptional bool
	var durationRequired bool
	var maxRequestDuration time.Duration
	dryRunResp, err := services.DryRunRequest(cmd.Context(), client, request)
	if err == nil {
		justificationOptional = services.IsJustificationOptionalForRequest(dryRunResp)
		durationRequired = services.IsDurationRequiredForRequest(dryRunResp)
		maxRequestDuration = services.GetMaximumRequestDuration(dryRunResp)
	}

	if accessDuration == nil && durationRequired {
		var newDuration *time.Duration
		newDuration, err = selectors.RunDurationInput(!durationRequired, 0, maxRequestDuration.Hours())
		if err != nil {
			return nil, err
		}

		accessDuration = newDuration
	}
	if accessDuration != nil {
		requestModels.Duration = accessDuration
		accessDurationInSec := int32(accessDuration.Seconds())
		request.DurationInSec = *clientapi.NewNullableInt32(&accessDurationInSec)
	}

	if justification == "" {
		var newJustification string
		newJustification, err = selectors.RunJustificationInput(justificationOptional)
		if err != nil {
			return nil, err
		}

		justification = newJustification
	}
	request.Justification = *clientapi.NewNullableString(&justification)

	err = GenerateAndPrintCreateRequestCommand(cmd, request, requestModels)
	if err != nil {
		return nil, err
	}

	return request, nil
}

func GenerateAndPrintCreateRequestCommand(cmd *cobra.Command, request *clientapi.CreateAccessRequestClientModel, models *CreateAccessRequestWithFullModels) error {
	if len(request.FilterBundleIds) != 0 {
		var bundleFlagValue string
		if models.Bundles != nil && len(models.Bundles) == 1 {
			bundleFlagValue = models.Bundles[0].Name
		} else {
			bundleFlagValue = request.FilterBundleIds[0]
		}

		return printCreateBundleRequestCommand(cmd, bundleFlagValue, request.Justification.Get(), models.Duration)
	}

	var integrationFlagValue string
	if models.Integrations != nil && len(models.Integrations) == 1 {
		integrationFlagValue = fmt.Sprintf("%s/%s", models.Integrations[0].Type, models.Integrations[0].Name)
	} else {
		integrationFlagValue = request.FilterIntegrationIds[0]
	}

	var resourcesFlagValues []string
	if models.Resources != nil {
		for _, resource := range models.Resources {
			resourcesFlagValues = append(resourcesFlagValues, resource.SourceId)
		}
	} else {
		for _, resourceFilter := range request.FilterResources {
			resourcesFlagValues = append(resourcesFlagValues, resourceFilter.Value)
		}
	}
	return printCreateIntegrationRequestCommand(cmd, integrationFlagValue, request.FilterResourceTypeIds[0], resourcesFlagValues, request.FilterPermissionIds, request.Justification.Get(), models.Duration)
}

func printCreateIntegrationRequestCommand(cmd *cobra.Command, integration string, resourceType string, resourceIDs []string, permissionIDs []string, justification *string, duration *time.Duration) error {
	createCommand := fmt.Sprintf("apono requests create --integration \"%s\" --resource-type \"%s\"", integration, resourceType)

	var permissionFlags []string
	for _, permissionID := range permissionIDs {
		permissionFlags = append(permissionFlags, fmt.Sprintf("--permissions \"%s\"", permissionID))
	}
	createCommand += " " + strings.Join(permissionFlags, " ")

	var resourceFlags []string
	for _, resourceID := range resourceIDs {
		resourceFlags = append(resourceFlags, fmt.Sprintf("--resources \"%s\"", resourceID))
	}
	createCommand += " " + strings.Join(resourceFlags, " ")

	if duration != nil {
		createCommand += fmt.Sprintf(" --duration %s", duration)
	}

	if justification != nil && *justification != "" {
		createCommand += fmt.Sprintf(" --justification \"%s\"", *justification)
	}

	err := printCreateCommand(cmd, createCommand)
	if err != nil {
		return err
	}

	return nil
}

func printCreateBundleRequestCommand(cmd *cobra.Command, bundle string, justification *string, duration *time.Duration) error {
	createCommand := fmt.Sprintf("apono requests create --bundle \"%s\"", bundle)
	if duration != nil {
		createCommand += fmt.Sprintf(" --duration %s", duration)
	}
	if justification != nil && *justification != "" {
		createCommand += fmt.Sprintf(" --justification \"%s\"", *justification)
	}

	err := printCreateCommand(cmd, createCommand)
	if err != nil {
		return err
	}

	return nil
}

func printCreateCommand(cmd *cobra.Command, commandString string) error {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "\n%s Use the following command to request this access again or create an alias for it: %s\n", styles.NoticeMsgPrefix, color.Green.Sprint(commandString))
	if err != nil {
		return err
	}

	return nil
}
