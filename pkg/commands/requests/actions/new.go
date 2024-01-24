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
	defaultWaitTimeForNewRequest = 60 * time.Second
)

type createRequestFlags struct {
	bundleIDOrName      string
	integrationIDOrName string
	resourceType        string
	resourceIDs         []string
	permissionIDs       []string
	justification       string
	runInteractiveMode  bool
	noWait              bool
	timeout             time.Duration
	output              utils.Format
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

			var newAccessRequest *clientapi.AccessRequestClientModel

			newAccessRequest, err = requestloader.RunRequestLoader(cmd.Context(), client, creationTime, cmdFlags.timeout, cmdFlags.noWait)
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
	req := services.GetEmptyNewRequestAPIModel()
	req.Justification = flags.justification

	switch {
	case flags.integrationIDOrName != "":
		var integration *clientapi.IntegrationClientModel
		integration, err := services.GetIntegrationByIDOrByTypeAndName(cmd.Context(), client, flags.integrationIDOrName)
		if err != nil {
			return nil, err
		}

		if flags.runInteractiveMode {
			req, err = flows.StartIntegrationRequestBuilderInteractiveMode(cmd, client, integration.Id, flags.resourceType, flags.resourceIDs, flags.permissionIDs, flags.justification)
			if err != nil {
				return nil, err
			}
		} else {
			resourceIds, err := listResourcesIDsFromSourceIDs(cmd.Context(), client, integration.Id, flags.resourceType, flags.resourceIDs)
			if err != nil {
				return nil, err
			}

			req.FilterIntegrationIds = []string{integration.Id}
			req.FilterResourceTypeIds = []string{flags.resourceType}
			req.FilterResourceIds = resourceIds
			req.FilterPermissionIds = flags.permissionIDs
		}

	case flags.bundleIDOrName != "":
		var bundle *clientapi.BundleClientModel
		bundle, err := services.GetBundleByNameOrID(cmd.Context(), client, flags.bundleIDOrName)
		if err != nil {
			return nil, err
		}

		if flags.runInteractiveMode {
			req, err = flows.StartBundleRequestBuilderInteractiveMode(cmd, client, bundle.Id, flags.justification)
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
