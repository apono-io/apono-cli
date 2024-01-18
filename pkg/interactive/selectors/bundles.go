package selectors

import (
	"context"
	"fmt"

	listselect "github.com/apono-io/apono-cli/pkg/interactive/inputs/list_select"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/services"
)

func RunBundleSelector(ctx context.Context, client *aponoapi.AponoClient) (*clientapi.BundleClientModel, error) {
	bundles, err := services.ListBundles(ctx, client, "")
	if err != nil {
		return nil, err
	}
	if len(bundles) == 0 {
		return nil, fmt.Errorf("no bundles found")
	}

	bundleByID := make(map[string]clientapi.BundleClientModel)
	var options []listselect.SelectOption
	for _, bundle := range bundles {
		options = append(options, listselect.SelectOption{
			ID:    bundle.Id,
			Label: bundle.Name,
		})
		bundleByID[bundle.Id] = bundle
	}

	bundleInput := listselect.SelectInput{
		Title:         "Select bundle",
		PostTitle:     "Selected bundle",
		Options:       options,
		ShowHelp:      true,
		EnableFilter:  true,
		ShowItemCount: true,
	}

	selectedItems, err := listselect.LaunchSelector(bundleInput)
	if err != nil {
		return nil, err
	}

	selectedBundle, ok := bundleByID[selectedItems[0].ID]
	if !ok {
		return nil, fmt.Errorf("bundle not found")
	}

	return &selectedBundle, nil
}
