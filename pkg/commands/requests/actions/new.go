package actions

import (
	"context"
	"fmt"
	"strings"
	"time"

	requestloader "github.com/apono-io/apono-cli/pkg/interactive/inputs/request_loader"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/interactive/flows"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/utils"

	"github.com/spf13/cobra"
)

const (
	bundleFlagName               = "bundle"
	integrationFlagName          = "integration"
	resourceTypeFlagName         = "resource-type"
	resourceFlagName             = "resources"
	permissionFlagName           = "permissions"
	justificationFlagName        = "justification"
	interactiveFlagName          = "interactive"
	noWaitFlagName               = "no-wait"
	timeoutFlagName              = "timeout"
	durationFlagName             = "duration"
	defaultWaitTimeForNewRequest = 60 * time.Second
	defaultAccessDuration        = 0
)

type createRequestFlags struct {
	bundleIDOrName      string
	integrationIDOrName string
	resourceType        string
	resourceIDs         []string
	permissionIDs       []string
	justification       string
	accessDuration      time.Duration
	runInteractiveMode  bool
	noWait              bool
	timeout             time.Duration
	output              utils.Format
	customFields        []string
}

func Create() *cobra.Command {
	cmdFlags := &createRequestFlags{}

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a new access request",
		Aliases: []string{"new"},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			req, err := createNewRequestAPIModelFromFlags(cmd, client, cmdFlags)
			if err != nil {
				return err
			}

			createResp, resp, err := client.ClientAPI.AccessRequestsAPI.CreateUserAccessRequest(cmd.Context()).
				CreateAccessRequestClientModel(*req).
				Execute()
			if err != nil {
				apiError := utils.ReturnAPIResponseError(resp)
				if apiError != nil {
					return apiError
				}

				return err
			}

			var newAccessRequest *clientapi.AccessRequestClientModel

			if len(createResp.RequestIds) == 0 {
				return fmt.Errorf("failed to create access request, no request IDs returned from the API")
			}
			requestID := createResp.RequestIds[0]
			newAccessRequest, err = waitForRequest(cmd.Context(), client, cmdFlags, requestID)
			if err != nil {
				return err
			}

			if cmdFlags.runInteractiveMode {
				fmt.Println()
			}

			err = services.PrintAccessRequests(cmd, []clientapi.AccessRequestClientModel{*newAccessRequest}, cmdFlags.output, false)
			if err != nil {
				return err
			}

			if services.IsRequestWaitingForMFA(newAccessRequest) && cmdFlags.output == utils.TableFormat {
				err = services.PrintAccessRequestMFALink(cmd, &newAccessRequest.Id)
				if err != nil {
					return err
				}
			}

			return nil
		},
	}

	flags := cmd.Flags()
	utils.AddFormatFlag(flags, &cmdFlags.output)
	flags.StringVarP(&cmdFlags.bundleIDOrName, bundleFlagName, "b", "", "The bundle id or name")
	flags.StringVarP(&cmdFlags.integrationIDOrName, integrationFlagName, "i", "", "The integration id or type/name, for example: \"aws-account/My AWS integration\"")
	flags.StringVarP(&cmdFlags.resourceType, resourceTypeFlagName, "t", "", "The resource type")
	flags.StringSliceVarP(&cmdFlags.resourceIDs, resourceFlagName, "r", []string{}, "The resource id's")
	flags.StringSliceVarP(&cmdFlags.permissionIDs, permissionFlagName, "p", []string{}, "The permission names")
	flags.StringVarP(&cmdFlags.justification, justificationFlagName, "j", "", "The justification for the access request")
	flags.BoolVar(&cmdFlags.runInteractiveMode, interactiveFlagName, false, "Run interactive mode")
	flags.BoolVar(&cmdFlags.noWait, noWaitFlagName, false, "Dont wait for the request to be granted")
	flags.DurationVar(&cmdFlags.timeout, timeoutFlagName, defaultWaitTimeForNewRequest, "Timeout for waiting for the request to be granted")
	flags.DurationVarP(&cmdFlags.accessDuration, durationFlagName, "d", defaultAccessDuration, "The duration of the access request")
	flags.StringSliceVar(&cmdFlags.customFields, "custom-field", []string{}, "Custom field values in format 'field-id=value'")

	cmd.MarkFlagsMutuallyExclusive(bundleFlagName, integrationFlagName)

	_ = cmd.RegisterFlagCompletionFunc(integrationFlagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return integrationsAutocompleteFunc(cmd, toComplete)
	})

	_ = cmd.RegisterFlagCompletionFunc(resourceTypeFlagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return resourceTypeAutocompleteFunc(cmd, cmdFlags.integrationIDOrName, toComplete)
	})

	_ = cmd.RegisterFlagCompletionFunc(resourceFlagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return resourcesAutocompleteFunc(cmd, cmdFlags.integrationIDOrName, cmdFlags.resourceType, toComplete)
	})

	_ = cmd.RegisterFlagCompletionFunc(permissionFlagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return permissionsAutocompleteFunc(cmd, cmdFlags.integrationIDOrName, cmdFlags.resourceType, toComplete)
	})

	_ = cmd.RegisterFlagCompletionFunc(bundleFlagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return bundlesAutoCompleteFunc(cmd, toComplete)
	})

	return cmd
}

func parseCustomFields(fields []string) (map[string]string, error) {
	result := make(map[string]string)

	for _, field := range fields {
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid custom field format: %s (expected 'field-id=value')", field)
		}

		fieldID := strings.TrimSpace(parts[0])
		fieldValue := strings.TrimSpace(parts[1])

		result[fieldID] = fieldValue
	}

	return result, nil
}

func createNewRequestAPIModelFromFlags(cmd *cobra.Command, client *aponoapi.AponoClient, flags *createRequestFlags) (*clientapi.CreateAccessRequestClientModel, error) {
	req := services.GetEmptyNewRequestAPIModel()

	if flags.justification != "" {
		req.Justification = *clientapi.NewNullableString(&flags.justification)
	}

	durationFlag := cmd.Flag(durationFlagName)
	var durationFlagValue *time.Duration
	if durationFlag.Changed {
		if flags.accessDuration <= 0 {
			return nil, fmt.Errorf("duration must be greater than 0")
		}

		durationFlagValue = &flags.accessDuration
		durationInSec := int32(flags.accessDuration.Seconds())
		req.DurationInSec = *clientapi.NewNullableInt32(&durationInSec)
	}

	customFieldValues, err := parseCustomFields(flags.customFields)
	if err != nil {
		return nil, err
	}

	req.CustomFields = customFieldValues

	switch {
	case flags.integrationIDOrName != "":
		err := validateIntegrationRequestFlagCombinations(flags)
		if err != nil {
			return nil, err
		}

		var integration *clientapi.IntegrationClientModel
		integration, err = services.GetIntegrationByIDOrByTypeAndName(cmd.Context(), client, flags.integrationIDOrName)
		if err != nil {
			return nil, err
		}

		if flags.runInteractiveMode {
			req, err = flows.StartIntegrationRequestBuilderInteractiveMode(cmd, client, integration.Id, flags.resourceType, flags.resourceIDs, flags.permissionIDs, flags.justification, durationFlagValue)
			if err != nil {
				return nil, err
			}
		} else {
			resourceIDs, err := listResourcesIDsFromSourceIDs(cmd.Context(), client, integration.Id, flags.resourceType, flags.resourceIDs)
			if err != nil {
				return nil, err
			}

			req.FilterIntegrationIds = []string{integration.Id}
			req.FilterResourceTypeIds = []string{flags.resourceType}
			req.FilterResources = services.ListResourceFiltersFromResourcesIDs(resourceIDs)
			req.FilterPermissionIds = flags.permissionIDs
		}

	case flags.bundleIDOrName != "":
		bundle, err := services.GetBundleByNameOrID(cmd.Context(), client, flags.bundleIDOrName)
		if err != nil {
			return nil, err
		}

		if flags.runInteractiveMode {
			req, err = flows.StartBundleRequestBuilderInteractiveMode(cmd, client, bundle.Id, flags.justification, durationFlagValue)
			if err != nil {
				return nil, err
			}
		} else {
			req.FilterBundleIds = []string{bundle.Id}
		}

	default:
		if flags.runInteractiveMode {
			var err error
			req, err = flows.StartRequestBuilderInteractiveMode(cmd, client)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("either --%s, --%s or --%s flags must be specified", integrationFlagName, bundleFlagName, interactiveFlagName)
		}
	}

	if !flags.runInteractiveMode {
		dryRunValidationErr := dryRunValidation(cmd, client, flags, req)
		if dryRunValidationErr != nil {
			return nil, dryRunValidationErr
		}
	}

	return req, nil
}

func integrationsAutocompleteFunc(cmd *cobra.Command, toComplete string) ([]string, cobra.ShellCompDirective) {
	return completeWithClient(cmd, func(client *aponoapi.AponoClient) ([]string, cobra.ShellCompDirective) {
		selectableIntegrations, err := services.ListIntegrations(cmd.Context(), client)
		if err != nil {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "failed to fetch available integrations:", err)
			return nil, cobra.ShellCompDirectiveError
		}

		integrationLabels := make(map[string]string)
		for _, val := range selectableIntegrations {
			integrationLabels[val.Id] = fmt.Sprintf("%s/%s", val.Type, val.Name)
		}

		extractor := func(val clientapi.IntegrationClientModel) string {
			return integrationLabels[val.Id]
		}

		return filterOptions[clientapi.IntegrationClientModel](selectableIntegrations, extractor, toComplete), cobra.ShellCompDirectiveDefault
	})
}

func resourceTypeAutocompleteFunc(cmd *cobra.Command, integrationIDOrName string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if integrationIDOrName == "" {
		return nil, cobra.ShellCompDirectiveError
	}

	return completeWithClient(cmd, func(client *aponoapi.AponoClient) ([]string, cobra.ShellCompDirective) {
		integration, err := services.GetIntegrationByIDOrByTypeAndName(cmd.Context(), client, integrationIDOrName)
		if err != nil {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "failed to fetch integration:", err)
			return nil, cobra.ShellCompDirectiveError
		}

		resp, err := services.ListResourceTypes(cmd.Context(), client, integration.Id)
		if err != nil {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "failed to fetch available resource types:", err)
			return nil, cobra.ShellCompDirectiveError
		}

		return filterOptions[clientapi.ResourceTypeClientModel](resp, func(val clientapi.ResourceTypeClientModel) string { return val.Id }, toComplete), cobra.ShellCompDirectiveDefault
	})
}

func resourcesAutocompleteFunc(cmd *cobra.Command, integrationIDOrName string, resourceType string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if integrationIDOrName == "" {
		return nil, cobra.ShellCompDirectiveError
	}

	if resourceType == "" {
		return nil, cobra.ShellCompDirectiveError
	}

	return completeWithClient(cmd, func(client *aponoapi.AponoClient) ([]string, cobra.ShellCompDirective) {
		integration, err := services.GetIntegrationByIDOrByTypeAndName(cmd.Context(), client, integrationIDOrName)
		if err != nil {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "failed to fetch integration:", err)
			return nil, cobra.ShellCompDirectiveError
		}

		resp, err := services.ListResources(cmd.Context(), client, integration.Id, resourceType, nil)
		if err != nil {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "failed to fetch available resources:", err)
			return nil, cobra.ShellCompDirectiveError
		}

		return filterOptions[clientapi.ResourceClientModel](resp, func(val clientapi.ResourceClientModel) string { return val.Id }, toComplete), cobra.ShellCompDirectiveDefault
	})
}

func permissionsAutocompleteFunc(cmd *cobra.Command, integrationIDOrName string, resourceType string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if integrationIDOrName == "" {
		return nil, cobra.ShellCompDirectiveError
	}

	if resourceType == "" {
		return nil, cobra.ShellCompDirectiveError
	}

	return completeWithClient(cmd, func(client *aponoapi.AponoClient) ([]string, cobra.ShellCompDirective) {
		integration, err := services.GetIntegrationByIDOrByTypeAndName(cmd.Context(), client, integrationIDOrName)
		if err != nil {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "failed to fetch integration:", err)
			return nil, cobra.ShellCompDirectiveError
		}

		resp, err := services.ListPermissions(cmd.Context(), client, integration.Id, resourceType)
		if err != nil {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "failed to fetch available permissions:", err)
			return nil, cobra.ShellCompDirectiveError
		}

		return filterOptions[clientapi.PermissionClientModel](resp, func(val clientapi.PermissionClientModel) string { return val.Id }, toComplete), cobra.ShellCompDirectiveDefault
	})
}

func bundlesAutoCompleteFunc(cmd *cobra.Command, toComplete string) ([]string, cobra.ShellCompDirective) {
	return completeWithClient(cmd, func(client *aponoapi.AponoClient) ([]string, cobra.ShellCompDirective) {
		selectableBundles, err := services.ListBundles(cmd.Context(), client, "")
		if err != nil {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "failed to fetch available bundles:", err)
			return nil, cobra.ShellCompDirectiveError
		}

		extractor := func(val clientapi.BundleClientModel) string {
			return val.Name
		}

		return filterOptions[clientapi.BundleClientModel](selectableBundles, extractor, toComplete), cobra.ShellCompDirectiveDefault
	})
}

func completeWithClient(cmd *cobra.Command, f func(client *aponoapi.AponoClient) ([]string, cobra.ShellCompDirective)) ([]string, cobra.ShellCompDirective) {
	client, err := aponoapi.GetClient(cmd.Context())
	if err != nil {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "failed to create Apono client:", err)
		return nil, cobra.ShellCompDirectiveError
	}

	return f(client)
}

func filterOptions[T any](allOptions []T, optionValueExtractor func(T) string, toComplete string) []string {
	var options []string
	for _, option := range allOptions {
		optionValue := optionValueExtractor(option)
		if strings.HasPrefix(strings.ToLower(optionValue), strings.ToLower(toComplete)) {
			options = append(options, optionValue)
		}
	}

	return options
}

func listResourcesIDsFromSourceIDs(ctx context.Context, client *aponoapi.AponoClient, integrationID string, resourceType string, resourceIDs []string) ([]string, error) {
	resources, err := services.ListResourcesBySourceIDs(ctx, client, integrationID, resourceType, resourceIDs)
	if err != nil {
		return nil, err
	}

	var resourceIDsToReturn []string
	for _, resource := range resources {
		resourceIDsToReturn = append(resourceIDsToReturn, resource.Id)
	}

	return resourceIDsToReturn, nil
}

func waitForRequest(ctx context.Context, client *aponoapi.AponoClient, cmdFlags *createRequestFlags, requestID string) (*clientapi.AccessRequestClientModel, error) {
	var newAccessRequest *clientapi.AccessRequestClientModel
	var err error

	if cmdFlags.runInteractiveMode {
		newAccessRequest, err = requestloader.RunRequestLoader(ctx, client, requestID, cmdFlags.timeout, cmdFlags.noWait)
		if err != nil {
			return nil, err
		}

		return newAccessRequest, nil
	}

	newAccessRequest, err = services.GetRequestByID(ctx, client, requestID)
	if err != nil {
		return nil, err
	}

	if cmdFlags.noWait {
		return newAccessRequest, nil
	}

	startTime := time.Now()
	for {
		if requestloader.ShouldStopLoading(newAccessRequest) {
			return newAccessRequest, nil
		}

		if time.Now().After(startTime.Add(cmdFlags.timeout)) {
			return newAccessRequest, fmt.Errorf("timeout waiting for request to be granted")
		}

		time.Sleep(1 * time.Second)

		newAccessRequest, err = services.GetRequestByID(ctx, client, requestID)
		if err != nil {
			return nil, err
		}
	}
}

func dryRunValidation(cmd *cobra.Command, client *aponoapi.AponoClient, flags *createRequestFlags, req *clientapi.CreateAccessRequestClientModel) error {
	dryRunResp, err := services.DryRunRequest(cmd.Context(), client, req)
	if err != nil {
		return nil
	}

	isJustificationOptional := services.IsJustificationOptionalForRequest(dryRunResp)
	if !isJustificationOptional && !req.Justification.IsSet() {
		return fmt.Errorf("justification is required for this request, please use the --%s flag", justificationFlagName)
	}

	isDurationRequired := services.IsDurationRequiredForRequest(dryRunResp)
	if isDurationRequired {
		if !req.DurationInSec.IsSet() {
			return fmt.Errorf("duration is required for this request, please use the --%s flag", durationFlagName)
		}

		requestMaximumDuration := services.GetMaximumRequestDuration(dryRunResp)
		if flags.accessDuration > requestMaximumDuration {
			return fmt.Errorf("duration is too long, maximum duration is %.2f hours", requestMaximumDuration.Hours())
		}
	}

	return nil
}

func validateIntegrationRequestFlagCombinations(flags *createRequestFlags) error {
	if !flags.runInteractiveMode {
		if flags.integrationIDOrName == "" || flags.resourceType == "" || len(flags.resourceIDs) == 0 || len(flags.permissionIDs) == 0 {
			return fmt.Errorf(
				"the following flags must be specified when requesting without interactive mode: --%s, --%s, --%s and --%s ",
				integrationFlagName, resourceTypeFlagName, resourceFlagName, permissionFlagName,
			)
		}
	}

	if flags.integrationIDOrName == "" && flags.resourceType != "" {
		return fmt.Errorf("flag --%s required when --%s is specified", integrationFlagName, resourceTypeFlagName)
	}

	if flags.resourceType == "" {
		if len(flags.resourceIDs) > 0 || len(flags.permissionIDs) > 0 {
			return fmt.Errorf("flag --%s required when one of the following flags are specified: --%s or --%s",
				resourceTypeFlagName, resourceFlagName, permissionFlagName,
			)
		}
	}

	return nil
}
