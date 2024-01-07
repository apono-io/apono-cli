package actions

import (
	"fmt"
	"strings"
	"time"

	"github.com/apono-io/apono-cli/pkg/services"

	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/utils"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

const (
	bundleFlagName           = "bundle"
	integrationFlagName      = "integration"
	resourceTypeFlagName     = "resource-type"
	resourceFlagName         = "resources"
	permissionFlagName       = "permissions"
	justificationFlagName    = "justification"
	maxWaitTimeForNewRequest = 30 * time.Second
)

func Create() *cobra.Command {
	req := clientapi.CreateAccessRequestClientModel{}
	req.FilterBundleIds = []string{}
	req.FilterAccessUnitIds = []string{}

	var integrationIDOrName string
	var bundleIDOrName string
	var resourceType string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new access request",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			switch {
			case integrationIDOrName != "":
				var integration *clientapi.IntegrationClientModel
				integration, err = services.GetIntegrationByIDOrByTypeAndName(cmd.Context(), client, integrationIDOrName)
				if err != nil {
					return err
				}

				req.FilterIntegrationIds = []string{integration.Id}
				req.FilterResourceTypeIds = []string{resourceType}
				req.FilterBundleIds = []string{}

			case bundleIDOrName != "":
				req.FilterIntegrationIds = []string{}
				req.FilterResourceTypeIds = []string{}

				var bundle *clientapi.BundleClientModel
				bundle, err = services.GetBundleByNameOrID(cmd.Context(), client, bundleIDOrName)
				if err != nil {
					return err
				}
				req.FilterBundleIds = []string{bundle.Id}

			default:
				return fmt.Errorf("either integration or bundle must be specified")
			}

			creationTime := time.Now()

			_, _, err = client.ClientAPI.AccessRequestsAPI.CreateUserAccessRequest(cmd.Context()).
				CreateAccessRequestClientModel(req).
				Execute()
			if err != nil {
				return err
			}

			newAccessRequest, err := waitForNewRequest(cmd, client, &creationTime)
			if err != nil {
				return err
			}

			table := services.GenerateRequestsTable([]clientapi.AccessRequestClientModel{*newAccessRequest})

			_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
			return err
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&bundleIDOrName, bundleFlagName, "b", "", "The bundle id or name")
	flags.StringVarP(&integrationIDOrName, integrationFlagName, "i", "", "The integration id or type/name, for example: \"aws-account/My AWS integration\"")
	flags.StringVarP(&resourceType, resourceTypeFlagName, "t", "", "The resource type")
	flags.StringSliceVarP(&req.FilterResourceIds, resourceFlagName, "r", []string{}, "The resource id's")
	flags.StringSliceVarP(&req.FilterPermissionIds, permissionFlagName, "p", []string{}, "The permission names")
	flags.StringVarP(&req.Justification, justificationFlagName, "j", "", "The justification for the access request")

	cmd.MarkFlagsRequiredTogether(integrationFlagName, resourceTypeFlagName, resourceFlagName, permissionFlagName)
	cmd.MarkFlagsMutuallyExclusive(bundleFlagName, integrationFlagName)
	_ = cmd.MarkFlagRequired(justificationFlagName)

	_ = cmd.RegisterFlagCompletionFunc(integrationFlagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return integrationsAutocompleteFunc(cmd, toComplete)
	})

	_ = cmd.RegisterFlagCompletionFunc(resourceTypeFlagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return resourceTypeAutocompleteFunc(cmd, integrationIDOrName, toComplete)
	})

	_ = cmd.RegisterFlagCompletionFunc(resourceFlagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return resourcesAutocompleteFunc(cmd, integrationIDOrName, resourceType, toComplete)
	})

	_ = cmd.RegisterFlagCompletionFunc(permissionFlagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return permissionsAutocompleteFunc(cmd, integrationIDOrName, resourceType, toComplete)
	})

	_ = cmd.RegisterFlagCompletionFunc(bundleFlagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return bundlesAutoCompleteFunc(cmd, toComplete)
	})

	return cmd
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
