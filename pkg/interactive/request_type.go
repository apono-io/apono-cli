package interactive

import (
	listselect "github.com/apono-io/apono-cli/pkg/interactive/inputs/list_select"
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
		Title:     "Select request type",
		PostTitle: "Selected request type",
		Options:   options,
	}

	selectedRequestTypes, err := listselect.LaunchSelector(requestTypeInput)
	if err != nil {
		return "", err
	}

	return selectedRequestTypes[0].ID, nil
}
