package interactive

import (
	listselect "github.com/apono-io/apono-cli/pkg/interactive/inputs/list_select"
	"github.com/apono-io/apono-cli/pkg/styles"
)

const (
	BundleRequestType      = "Bundle"
	IntegrationRequestType = "Integration"
)

func RunRequestTypeSelector() (string, error) {
	options := []listselect.SelectOption{
		{
			ID:    BundleRequestType,
			Label: BundleRequestType,
		},
		{
			ID:    IntegrationRequestType,
			Label: IntegrationRequestType,
		},
	}

	requestTypeInput := listselect.SelectInput{
		Title:   styles.BeforeSelectingItemsTitleStyle("Select request type"),
		Options: options,
		PostMessage: func(s []listselect.SelectOption) string {
			return styles.AfterSelectingItemsTitleStyle("Selected request type", []string{s[0].Label})
		},
	}

	selectedRequestTypes, err := listselect.LaunchSelector(requestTypeInput)
	if err != nil {
		return "", err
	}

	return selectedRequestTypes[0].ID, nil
}
