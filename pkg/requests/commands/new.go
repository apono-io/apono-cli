package commands

import (
	"fmt"
	"strings"

	"github.com/apono-io/apono-sdk-go"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

const (
	integrationFlagName   = "integration"
	resourceTypeFlagName  = "resource-type"
	resourceFlagName      = "resource"
	permissionFlagName    = "permission"
	justificationFlagName = "justification"
)

func New() *cobra.Command {
	req := apono.CreateAccessRequest{}
	var resourceType string
	cmd := &cobra.Command{
		Use:     "request",
		GroupID: Group.ID,
		Short:   "New access request",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			req.UserId = client.Session.UserID
			accessRequest, _, err := client.AccessRequestsApi.CreateAccessRequest(cmd.Context()).
				CreateAccessRequest(req).
				Execute()
			if err != nil {
				return err
			}

			return printAccessRequestDetails(cmd, client, accessRequest)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&req.IntegrationId, integrationFlagName, "i", "", "integration id or name")
	flags.StringVarP(&resourceType, resourceTypeFlagName, "t", "", "resource type")
	flags.StringSliceVarP(&req.ResourceIds, resourceFlagName, "r", []string{}, "resource id")
	flags.StringSliceVarP(&req.Permissions, permissionFlagName, "p", []string{}, "permission name")
	flags.StringVarP(&req.Justification, justificationFlagName, "j", "", justificationFlagName)
	_ = cmd.MarkFlagRequired(integrationFlagName)
	_ = cmd.MarkFlagRequired(resourceTypeFlagName)
	_ = cmd.MarkFlagRequired(resourceFlagName)
	_ = cmd.MarkFlagRequired(permissionFlagName)
	_ = cmd.MarkFlagRequired(justificationFlagName)

	_ = cmd.RegisterFlagCompletionFunc(integrationFlagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return integrationsAutocompleteFunc(cmd, toComplete)
	})

	_ = cmd.RegisterFlagCompletionFunc(resourceTypeFlagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return resourceTypeAutocompleteFunc(cmd, req.IntegrationId, toComplete)
	})

	_ = cmd.RegisterFlagCompletionFunc(resourceFlagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return resourcesAutocompleteFunc(cmd, req.IntegrationId, resourceType, toComplete)
	})

	_ = cmd.RegisterFlagCompletionFunc(permissionFlagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return permissionsAutocompleteFunc(cmd, req.IntegrationId, resourceType, toComplete)
	})

	return cmd
}

func integrationsAutocompleteFunc(cmd *cobra.Command, toComplete string) ([]string, cobra.ShellCompDirective) {
	return completeWithClient(cmd, func(client *aponoapi.AponoClient) ([]string, cobra.ShellCompDirective) {
		selectableIntegrationsResp, _, err := client.AccessRequestsApi.GetSelectableIntegrations(cmd.Context()).Execute()
		if err != nil {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "failed to fetch selectable integrations:", err)
			return nil, cobra.ShellCompDirectiveError
		}

		resp, _, err := client.IntegrationsApi.ListIntegrationsV2(cmd.Context()).Execute()
		if err != nil {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "failed to fetch integrations:", err)
			return nil, cobra.ShellCompDirectiveError
		}

		integrationLabels := make(map[string]string)
		for _, val := range resp.Data {
			integrationLabels[val.Id] = fmt.Sprintf("%s/%s", val.Type, val.Name)
		}

		extractor := func(val apono.SelectableIntegration) string {
			return integrationLabels[val.Id]
		}

		return filterOptions[apono.SelectableIntegration](selectableIntegrationsResp.Data, extractor, toComplete), cobra.ShellCompDirectiveDefault
	})
}

func resourceTypeAutocompleteFunc(cmd *cobra.Command, integrationID string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if integrationID == "" {
		return nil, cobra.ShellCompDirectiveError
	}

	return completeWithClient(cmd, func(client *aponoapi.AponoClient) ([]string, cobra.ShellCompDirective) {
		resp, _, err := client.AccessRequestsApi.GetSelectableResourceTypes(cmd.Context(), integrationID).Execute()
		if err != nil {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "failed to fetch available resource types:", err)
			return nil, cobra.ShellCompDirectiveError
		}

		return filterOptions[apono.SelectableResourceType](resp.Data, func(val apono.SelectableResourceType) string { return val.Id }, toComplete), cobra.ShellCompDirectiveDefault
	})
}

func resourcesAutocompleteFunc(cmd *cobra.Command, integrationID string, resourceType string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if integrationID == "" {
		return nil, cobra.ShellCompDirectiveError
	}

	if resourceType == "" {
		return nil, cobra.ShellCompDirectiveError
	}

	return completeWithClient(cmd, func(client *aponoapi.AponoClient) ([]string, cobra.ShellCompDirective) {
		resp, _, err := client.AccessRequestsApi.GetSelectableResources(cmd.Context(), integrationID, resourceType).Execute()
		if err != nil {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "failed to fetch available resources:", err)
			return nil, cobra.ShellCompDirectiveError
		}

		return filterOptions[apono.SelectableResource](resp.Data, func(val apono.SelectableResource) string { return val.Id }, toComplete), cobra.ShellCompDirectiveDefault
	})
}

func permissionsAutocompleteFunc(cmd *cobra.Command, integrationID string, resourceType string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if integrationID == "" {
		return nil, cobra.ShellCompDirectiveError
	}

	if resourceType == "" {
		return nil, cobra.ShellCompDirectiveError
	}

	return completeWithClient(cmd, func(client *aponoapi.AponoClient) ([]string, cobra.ShellCompDirective) {
		resp, _, err := client.AccessRequestsApi.GetSelectablePermissions(cmd.Context(), integrationID, resourceType).Execute()
		if err != nil {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "failed to fetch available permissions:", err)
			return nil, cobra.ShellCompDirectiveError
		}

		return filterOptions[string](resp.Data, func(val string) string { return val }, toComplete), cobra.ShellCompDirectiveDefault
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
