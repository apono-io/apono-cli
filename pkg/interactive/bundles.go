package interactive

import (
	"context"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	listselect "github.com/apono-io/apono-cli/pkg/inputs/list_select"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/styles"
)

func RunBundleSelector(ctx context.Context, client *aponoapi.AponoClient) (*clientapi.BundleClientModel, error) {
	bundles, err := services.ListBundles(ctx, client)
	if err != nil {
		return nil, err
	}

	bundleInput := listselect.SelectInput[clientapi.BundleClientModel]{
		Title:       styles.BeforeSelectingItemsTitleStyle("Select bundle"),
		Options:     bundles,
		FilterFunc:  func(s clientapi.BundleClientModel) string { return s.Name },
		DisplayFunc: func(s clientapi.BundleClientModel) string { return s.Name },
		IsEqual:     func(s clientapi.BundleClientModel, s2 clientapi.BundleClientModel) bool { return s.Id == s2.Id },
		PostMessage: func(s []clientapi.BundleClientModel) string {
			return styles.AfterSelectingItemsTitleStyle("Selected bundle", []string{s[0].Name})
		},
		ShowHelp:      true,
		EnableFilter:  true,
		ShowItemCount: true,
	}

	selectedBundles, err := listselect.LaunchSelector(bundleInput)
	if err != nil {
		return nil, err
	}

	return &selectedBundles[0], nil
}
