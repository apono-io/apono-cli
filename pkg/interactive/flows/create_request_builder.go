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

	requestCustomFields, err := services.GetRequestCustomFields(cmd.Context(), client)
	if err != nil {
		return nil, err
	}

	customFieldValues, err := selectors.RunCustomFieldsInputs(requestCustomFields)
	if err != nil {
		return nil, err
	}

	request.CustomFields = customFieldValues

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

	integration, err := resolveIntegration(cmd, client, integrationID)
	if err != nil {
		return nil, err
	}

	resourceType, err := resolveResourceType(cmd, client, integration.Id, resourceTypeID)
	if err != nil {
		return nil, err
	}

	resources, err := resolveResources(cmd, client, integration.Id, resourceType.Id, resourceIDs)
	if err != nil {
		return nil, err
	}
	var resolvedResourceIDs []string
	for _, resource := range resources {
		resolvedResourceIDs = append(resolvedResourceIDs, resource.Id)
	}

	permissions, err := resolvePermissions(cmd, client, integration.Id, resourceType.Id, permissionIDs, resourceType.AllowMultiplePermissions)
	if err != nil {
		return nil, err
	}

	requestModels.Integrations = []clientapi.IntegrationClientModel{*integration}
	requestModels.Resources = resources

	request.FilterIntegrationIds = []string{integration.Id}
	request.FilterResourceTypeIds = []string{resourceType.Id}
	request.FilterResources = services.ListResourceFiltersFromResourcesIDs(resolvedResourceIDs)
	request.FilterPermissionIds = permissions

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
		accessDuration, err = selectors.RunDurationInput(!durationRequired, 0, maxRequestDuration.Hours())
		if err != nil {
			return nil, err
		}
	}
	if accessDuration != nil {
		requestModels.Duration = accessDuration
		accessDurationInSec := int32(accessDuration.Seconds())
		request.DurationInSec = *clientapi.NewNullableInt32(&accessDurationInSec)
	}

	resolvedJustification, err := resolveJustification(justification, justificationOptional)
	if err != nil {
		return nil, err
	}
	request.Justification = *clientapi.NewNullableString(resolvedJustification)

	requestCustomFields, err := services.GetRequestCustomFields(cmd.Context(), client)
	if err != nil {
		return nil, err
	}

	customFieldValues, err := selectors.RunCustomFieldsInputs(requestCustomFields)
	if err != nil {
		return nil, err
	}

	request.CustomFields = customFieldValues

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

		return printCreateBundleRequestCommand(cmd, bundleFlagValue, request.Justification.Get(), models.Duration, request.CustomFields)
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
	return printCreateIntegrationRequestCommand(cmd, integrationFlagValue, request.FilterResourceTypeIds[0], resourcesFlagValues, request.FilterPermissionIds, request.Justification.Get(), models.Duration, request.CustomFields)
}

func resolveIntegration(cmd *cobra.Command, client *aponoapi.AponoClient, integrationID string) (*clientapi.IntegrationClientModel, error) {
	if integrationID == "" {
		return selectors.RunIntegrationSelector(cmd.Context(), client)
	}

	return services.GetIntegrationByIDOrByTypeAndName(cmd.Context(), client, integrationID)
}

func resolveResourceType(cmd *cobra.Command, client *aponoapi.AponoClient, integrationID string, resourceTypeID string) (*clientapi.ResourceTypeClientModel, error) {
	if resourceTypeID == "" {
		return selectors.RunResourceTypeSelector(cmd.Context(), client, integrationID)
	}

	return services.GetResourceTypeByID(cmd.Context(), client, integrationID, resourceTypeID)
}

func resolveResources(cmd *cobra.Command, client *aponoapi.AponoClient, integrationID string, resourceTypeID string, resourceIDs []string) ([]clientapi.ResourceClientModel, error) {
	if len(resourceIDs) == 0 {
		return selectors.RunResourcesSelector(cmd.Context(), client, integrationID, resourceTypeID)
	}

	return services.ListResourcesBySourceIDs(cmd.Context(), client, integrationID, resourceTypeID, resourceIDs)
}

func resolvePermissions(cmd *cobra.Command, client *aponoapi.AponoClient, integrationID string, resourceTypeID string, permissionIDs []string, allowMultiplePermissions bool) ([]string, error) {
	if len(permissionIDs) == 0 {
		permissions, err := selectors.RunPermissionsSelector(cmd.Context(), client, integrationID, resourceTypeID, allowMultiplePermissions)
		if err != nil {
			return nil, err
		}

		var selectorPermissionIDs []string
		for _, permission := range permissions {
			selectorPermissionIDs = append(selectorPermissionIDs, permission.Id)
		}

		return selectorPermissionIDs, nil
	}

	if !allowMultiplePermissions && len(permissionIDs) > 1 {
		return nil, fmt.Errorf("only one permission can be selected for this resource type")
	}

	return permissionIDs, nil
}

func resolveJustification(userJustification string, isJustificationOptional bool) (*string, error) {
	var result string
	if userJustification == "" {
		selectorJustification, err := selectors.RunJustificationInput(isJustificationOptional)
		if err != nil {
			return nil, err
		}

		result = selectorJustification
	} else {
		result = userJustification
	}

	if result == "" {
		return nil, nil
	}

	return &result, nil
}

func printCreateIntegrationRequestCommand(cmd *cobra.Command, integration string, resourceType string, resourceIDs []string, permissionIDs []string, justification *string, duration *time.Duration, customFields map[string]string) error {
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

	for id, value := range customFields {
		createCommand += fmt.Sprintf(" --custom-field \"%s=%s\"", id, value)
	}

	err := printCreateCommand(cmd, createCommand)
	if err != nil {
		return err
	}

	return nil
}

func printCreateBundleRequestCommand(cmd *cobra.Command, bundle string, justification *string, duration *time.Duration, customFields map[string]string) error {
	createCommand := fmt.Sprintf("apono requests create --bundle \"%s\"", bundle)
	if duration != nil {
		createCommand += fmt.Sprintf(" --duration %s", duration)
	}
	if justification != nil && *justification != "" {
		createCommand += fmt.Sprintf(" --justification \"%s\"", *justification)
	}

	for id, value := range customFields {
		createCommand += fmt.Sprintf(" --custom-field \"%s=%s\"", id, value)
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
