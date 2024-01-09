package interactive

import (
	"github.com/apono-io/apono-cli/pkg/inputs/list_select"
	"github.com/apono-io/apono-cli/pkg/styles"
)

const (
	BundleRequestType      = "Bundle"
	IntegrationRequestType = "Integration"
)

func RunRequestTypeSelector() (string, error) {
	requestTypeInput := list_select.SelectInput[string]{
		Title:       styles.BeforeSelectingItemsTitleStyle("Select request type"),
		Options:     []string{BundleRequestType, IntegrationRequestType},
		FilterFunc:  func(s string) string { return s },
		DisplayFunc: func(s string) string { return s },
		IsEqual:     func(s string, s2 string) bool { return s == s2 },
		PostMessage: func(s []string) string {
			return styles.AfterSelectingItemsTitleStyle("Selected request type", []string{s[0]})
		},
	}

	selectedRequestTypes, err := list_select.LaunchSelector(requestTypeInput)
	if err != nil {
		return "", err
	}

	return selectedRequestTypes[0], nil
}
