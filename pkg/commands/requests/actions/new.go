package actions

import (
	"fmt"
	"strings"
	"time"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/interactive"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/utils"

	"github.com/spf13/cobra"
)

const (
	bundleFlagName           = "bundle"
	integrationFlagName      = "integration"
	resourceTypeFlagName     = "resource-type"
	resourceFlagName         = "resources"
	permissionFlagName       = "permissions"
	justificationFlagName    = "justification"
	noInteractiveFlagName    = "no-interactive"
	maxWaitTimeForNewRequest = 30 * time.Second
)

type createRequestFlags struct {
	bundleIDOrName      string
	integrationIDOrName string
	resourceType        string
	resourceIDs         []string
	permissionIDs       []string
	justification       string
	dontRunInteractive  bool
}

func Create() *cobra.Command {
	cmdFlags := &createRequestFlags{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new access request",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			req, err := createNewRequestAPIModelFromFlags(cmd, client, cmdFlags)
			if err != nil {
				return err
			}

			creationTime := time.Now()

			_, resp, err := client.ClientAPI.AccessRequestsAPI.CreateUserAccessRequest(cmd.Context()).
				CreateAccessRequestClientModel(*req).
				Execute()
			if err != nil {
				apiError := utils.ReturnAPIResponseError(resp)
				if apiError != nil {
					return apiError
				}

				return err
			}

			newAccessRequest, err := waitForNewRequest(cmd, client, &creationTime)
			if err != nil {
				return err
			}

			table := services.GenerateRequestsTable([]clientapi.AccessRequestClientModel{*newAccessRequest})

			_, err = fmt.Fprintln(cmd.OutOrStdout(), "")
			if err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
			return err
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&cmdFlags.bundleIDOrName, bundleFlagName, "b", "", "The bundle id or name")
	flags.StringVarP(&cmdFlags.integrationIDOrName, integrationFlagName, "i", "", "The integration id or type/name, for example: \"aws-account/My AWS integration\"")
	flags.StringVarP(&cmdFlags.resourceType, resourceTypeFlagName, "t", "", "The resource type")
	flags.StringSliceVarP(&cmdFlags.resourceIDs, resourceFlagName, "r", []string{}, "The resource id's")
	flags.StringSliceVarP(&cmdFlags.permissionIDs, permissionFlagName, "p", []string{}, "The permission names")
	flags.StringVarP(&cmdFlags.justification, justificationFlagName, "j", "", "The justification for the access request")
	flags.BoolVar(&cmdFlags.dontRunInteractive, noInteractiveFlagName, false, "Dont run in interactive mode")

	cmd.MarkFlagsRequiredTogether(integrationFlagName, resourceTypeFlagName, resourceFlagName, permissionFlagName)
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

func createNewRequestAPIModelFromFlags(cmd *cobra.Command, client *aponoapi.AponoClient, flags *createRequestFlags) (*clientapi.CreateAccessRequestClientModel, error) {
	req := getEmptyRequest()
	req.Justification = flags.justification

	switch {
	case flags.integrationIDOrName != "":
		var integration *clientapi.IntegrationClientModel
		integration, err := services.GetIntegrationByIDOrByTypeAndName(cmd.Context(), client, flags.integrationIDOrName)
		if err != nil {
			return nil, err
		}

		if flags.dontRunInteractive {
			req.FilterIntegrationIds = []string{integration.Id}
			req.FilterResourceTypeIds = []string{flags.resourceType}
			req.FilterResourceIds = flags.resourceIDs
			req.FilterPermissionIds = flags.permissionIDs
		} else {
			req, err = startIntegrationRequestInteractiveMode(cmd, client, integration.Id, flags.resourceType, flags.resourceIDs, flags.permissionIDs, flags.justification)
			if err != nil {
				return nil, err
			}
		}

	case flags.bundleIDOrName != "":
		var bundle *clientapi.BundleClientModel
		bundle, err := services.GetBundleByNameOrID(cmd.Context(), client, flags.bundleIDOrName)
		if err != nil {
			return nil, err
		}

		if flags.dontRunInteractive {
			req.FilterBundleIds = []string{bundle.Id}
		} else {
			req, err = startBundleRequestInteractiveMode(cmd, client, bundle.Id, flags.justification)
			if err != nil {
				return nil, err
			}
		}

	default:
		if flags.dontRunInteractive {
			return nil, fmt.Errorf("either integration or bundle must be specified")
		} else {
			var err error
			req, err = startRequestInteractiveMode(cmd, client)
			if err != nil {
				return nil, err
			}
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

//nolint:dupl // Remove duplication error
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

		resp, err := services.ListResources(cmd.Context(), client, integration.Id, resourceType)
		if err != nil {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "failed to fetch available resources:", err)
			return nil, cobra.ShellCompDirectiveError
		}

		return filterOptions[clientapi.ResourceClientModel](resp, func(val clientapi.ResourceClientModel) string { return val.Id }, toComplete), cobra.ShellCompDirectiveDefault
	})
}

//nolint:dupl // Remove duplication error
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

func waitForNewRequest(cmd *cobra.Command, client *aponoapi.AponoClient, creationTime *time.Time) (*clientapi.AccessRequestClientModel, error) {
	startTime := time.Now()
	for {
		lastRequest, err := services.GetUserLastRequest(cmd.Context(), client)
		if err != nil {
			return nil, err
		}

		lastRequestCreationTime := utils.ConvertUnixTimeToTime(lastRequest.CreationTime)
		if lastRequestCreationTime.After(*creationTime) {
			return lastRequest, nil
		}

		time.Sleep(1 * time.Second)

		if time.Now().After(startTime.Add(maxWaitTimeForNewRequest)) {
			return nil, fmt.Errorf("timeout while waiting for request to be created")
		}
	}
}

func startRequestInteractiveMode(cmd *cobra.Command, client *aponoapi.AponoClient) (*clientapi.CreateAccessRequestClientModel, error) {
	requestType, err := interactive.RunRequestTypeSelector()
	if err != nil {
		return nil, err
	}
	switch requestType {
	case interactive.BundleRequestType:
		return startBundleRequestInteractiveMode(cmd, client, "", "")
	case interactive.IntegrationRequestType:
		return startIntegrationRequestInteractiveMode(cmd, client, "", "", []string{}, []string{}, "")
	default:
		return nil, fmt.Errorf("invalid request type: %s", requestType)
	}
}

func startBundleRequestInteractiveMode(cmd *cobra.Command, client *aponoapi.AponoClient, bundleID string, justification string) (*clientapi.CreateAccessRequestClientModel, error) {
	request := getEmptyRequest()

	if bundleID == "" {
		bundle, err := interactive.RunBundleSelector(cmd.Context(), client)
		if err != nil {
			return nil, err
		}

		bundleID = bundle.Id
	}
	request.FilterBundleIds = []string{bundleID}

	if justification == "" {
		newJustification, err := interactive.RunJustificationInput()
		if err != nil {
			return nil, err
		}

		justification = newJustification
	}
	request.Justification = justification

	return request, nil
}

func startIntegrationRequestInteractiveMode(
	cmd *cobra.Command,
	client *aponoapi.AponoClient,
	integrationID string,
	resourceTypeID string,
	resourceIDs []string,
	permissionIDs []string,
	justification string,
) (*clientapi.CreateAccessRequestClientModel, error) {
	request := getEmptyRequest()

	if integrationID == "" {
		integration, err := interactive.RunIntegrationSelector(cmd.Context(), client)
		if err != nil {
			return nil, err
		}

		integrationID = integration.Id
	}
	request.FilterIntegrationIds = []string{integrationID}

	var allowMultiplePermissions bool
	if resourceTypeID == "" {
		resourceType, err := interactive.RunResourceTypeSelector(cmd.Context(), client, integrationID)
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
		resources, err := interactive.RunResourcesSelector(cmd.Context(), client, integrationID, resourceTypeID)
		if err != nil {
			return nil, err
		}

		for _, resource := range resources {
			resourceIDs = append(resourceIDs, resource.Id)
		}
	}
	request.FilterResourceIds = resourceIDs

	if len(permissionIDs) == 0 {
		permissions, err := interactive.RunPermissionsSelector(cmd.Context(), client, integrationID, resourceTypeID, allowMultiplePermissions)
		if err != nil {
			return nil, err
		}

		for _, permission := range permissions {
			permissionIDs = append(permissionIDs, permission.Id)
		}
	}
	request.FilterPermissionIds = permissionIDs

	if justification == "" {
		newJustification, err := interactive.RunJustificationInput()
		if err != nil {
			return nil, err
		}

		justification = newJustification
	}
	request.Justification = justification

	return request, nil
}

func getEmptyRequest() *clientapi.CreateAccessRequestClientModel {
	return &clientapi.CreateAccessRequestClientModel{
		FilterBundleIds:       []string{},
		FilterIntegrationIds:  []string{},
		FilterResourceTypeIds: []string{},
		FilterResourceIds:     []string{},
		FilterPermissionIds:   []string{},
		FilterAccessUnitIds:   []string{},
	}
}
