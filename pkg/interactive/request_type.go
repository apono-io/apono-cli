package interactive

import (
	listselect2 "github.com/apono-io/apono-cli/pkg/interactive/inputs/list_select"
	"github.com/apono-io/apono-cli/pkg/styles"
)

const (
	BundleRequestType      = "Bundle"
	IntegrationRequestType = "Integration"
)

func RunRequestTypeSelector() (string, error) {
	options := []listselect2.SelectOption{
		{
			ID:    BundleRequestType,
			Label: BundleRequestType,
		},
		{
			ID:    IntegrationRequestType,
			Label: IntegrationRequestType,
		},
	}

	requestTypeInput := listselect2.SelectInput{
		Title:   styles.BeforeSelectingItemsTitleStyle("Select request type"),
		Options: options,
		PostMessage: func(s []listselect2.SelectOption) string {
			return styles.AfterSelectingItemsTitleStyle("Selected request type", []string{s[0].Label})
		},
	}

	selectedRequestTypes, err := listselect2.LaunchSelector(requestTypeInput)
	if err != nil {
		return "", err
	}

	return selectedRequestTypes[0].ID, nil
}
