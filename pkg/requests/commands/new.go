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
	var (
		integration   string
		resourceIDs   []string
		permissions   []string
		justification string
	)

	cmd := &cobra.Command{
		Use:     "request",
		GroupID: Group.ID,
		Short:   "New access request",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.CreateClient(cmd.Context(), "default")
			if err != nil {
				return err
			}

			resp, err := client.CreateAccessRequestWithResponse(cmd.Context(), aponoapi.CreateAccessRequest{
				IntegrationId: integration,
				Justification: justification,
				Permissions:   permissions,
				ResourceIds:   resourceIDs,
				UserId:        client.Session.UserID,
			})
			if err != nil {
				return err
			}

			accessRequest := resp.JSON200
			return printAccessRequestDetails(cmd, client, accessRequest)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&integration, integrationFlagName, "i", "", "integration id or name")
	flags.StringSliceVarP(&resourceIDs, resourceFlagName, "r", []string{}, "resource id")
	flags.StringSliceVarP(&permissions, permissionFlagName, "p", []string{}, "permission name")
	flags.StringVarP(&justification, justificationFlagName, "j", "", justificationFlagName)
	_ = cmd.MarkFlagRequired(integrationFlagName)
	_ = cmd.MarkFlagRequired(resourceFlagName)
	_ = cmd.MarkFlagRequired(permissionFlagName)
	_ = cmd.MarkFlagRequired(justificationFlagName)

	_ = cmd.RegisterFlagCompletionFunc(integrationFlagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeWithClient(cmd, func(client *aponoapi.AponoClient) ([]string, cobra.ShellCompDirective) {
			resp, err := client.GetSelectableIntegrationsWithResponse(cmd.Context(), &aponoapi.GetSelectableIntegrationsParams{})
			if err != nil {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "failed to fetch available integrations:", err)
				return nil, cobra.ShellCompDirectiveError
			}

			return filterOptions[aponoapi.SelectableIntegration](resp.JSON200.Data, func(val aponoapi.SelectableIntegration) string { return val.Id }, toComplete), cobra.ShellCompDirectiveDefault
		})
	})

	_ = cmd.RegisterFlagCompletionFunc(resourceFlagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if integration == "" {
			return nil, cobra.ShellCompDirectiveError
		}

		return completeWithClient(cmd, func(client *aponoapi.AponoClient) ([]string, cobra.ShellCompDirective) {
			resp, err := client.GetSelectableResourcesWithResponse(cmd.Context(), integration, &aponoapi.GetSelectableResourcesParams{})
			if err != nil {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "failed to fetch available resources:", err)
				return nil, cobra.ShellCompDirectiveError
			}

			return filterOptions[aponoapi.SelectableResource](resp.JSON200.Data, func(val aponoapi.SelectableResource) string { return val.Id }, toComplete), cobra.ShellCompDirectiveDefault
		})
	})

	_ = cmd.RegisterFlagCompletionFunc(permissionFlagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if integration == "" {
			return nil, cobra.ShellCompDirectiveError
		}

		return completeWithClient(cmd, func(client *aponoapi.AponoClient) ([]string, cobra.ShellCompDirective) {
			resp, err := client.GetSelectablePermissionsWithResponse(cmd.Context(), integration, &aponoapi.GetSelectablePermissionsParams{})
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
	client, err := aponoapi.CreateClient(cmd.Context(), "default")
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
