package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

const (
	integrationFlagName   = "integration"
	resourceFlagName      = "resource"
	permissionFlagName    = "permission"
	justificationFlagName = "justification"
)

func New() *cobra.Command {
	req := aponoapi.CreateAccessRequest{}
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
			resp, err := client.CreateAccessRequestWithResponse(cmd.Context(), req)
			if err != nil {
				return err
			}

			accessRequest := resp.JSON200
			return printAccessRequestDetails(cmd, client, accessRequest)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&req.IntegrationId, integrationFlagName, "i", "", "integration id or name")
	flags.StringSliceVarP(&req.ResourceIds, resourceFlagName, "r", []string{}, "resource id")
	flags.StringSliceVarP(&req.Permissions, permissionFlagName, "p", []string{}, "permission name")
	flags.StringVarP(&req.Justification, justificationFlagName, "j", "", justificationFlagName)
	_ = cmd.MarkFlagRequired(integrationFlagName)
	_ = cmd.MarkFlagRequired(resourceFlagName)
	_ = cmd.MarkFlagRequired(permissionFlagName)
	_ = cmd.MarkFlagRequired(justificationFlagName)

	_ = cmd.RegisterFlagCompletionFunc(integrationFlagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeWithClient(cmd, func(client *aponoapi.AponoClient) ([]string, cobra.ShellCompDirective) {
			selectableIntegrationsResp, err := client.GetSelectableIntegrationsWithResponse(cmd.Context(), &aponoapi.GetSelectableIntegrationsParams{})
			if err != nil {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "failed to fetch selectable integrations:", err)
				return nil, cobra.ShellCompDirectiveError
			}

			resp, err := client.ListIntegrationsV2WithResponse(cmd.Context())
			if err != nil {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "failed to fetch integrations:", err)
				return nil, cobra.ShellCompDirectiveError
			}

			integrationLabels := make(map[string]string)
			for _, val := range resp.JSON200.Data {
				integrationLabels[val.Id] = fmt.Sprintf("%s/%s", val.Type, val.Name)
			}

			extractor := func(val aponoapi.SelectableIntegration) string {
				return integrationLabels[val.Id]
			}

			return filterOptions[aponoapi.SelectableIntegration](selectableIntegrationsResp.JSON200.Data, extractor, toComplete), cobra.ShellCompDirectiveDefault
		})
	})

	_ = cmd.RegisterFlagCompletionFunc(resourceFlagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if req.IntegrationId == "" {
			return nil, cobra.ShellCompDirectiveError
		}

		return completeWithClient(cmd, func(client *aponoapi.AponoClient) ([]string, cobra.ShellCompDirective) {
			resp, err := client.GetSelectableResourcesWithResponse(cmd.Context(), req.IntegrationId, &aponoapi.GetSelectableResourcesParams{})
			if err != nil {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "failed to fetch available resources:", err)
				return nil, cobra.ShellCompDirectiveError
			}

			return filterOptions[aponoapi.SelectableResource](resp.JSON200.Data, func(val aponoapi.SelectableResource) string { return val.Id }, toComplete), cobra.ShellCompDirectiveDefault
		})
	})

	_ = cmd.RegisterFlagCompletionFunc(permissionFlagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if req.IntegrationId == "" {
			return nil, cobra.ShellCompDirectiveError
		}

		return completeWithClient(cmd, func(client *aponoapi.AponoClient) ([]string, cobra.ShellCompDirective) {
			resp, err := client.GetSelectablePermissionsWithResponse(cmd.Context(), req.IntegrationId, &aponoapi.GetSelectablePermissionsParams{})
			if err != nil {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "failed to fetch available permissions:", err)
				return nil, cobra.ShellCompDirectiveError
			}

			return filterOptions[string](resp.JSON200.Data, func(val string) string { return val }, toComplete), cobra.ShellCompDirectiveDefault
		})
	})

	return cmd
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
