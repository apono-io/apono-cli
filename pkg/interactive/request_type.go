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
	requestTypeInput := listselect2.SelectInput[string]{
		Title:       styles.BeforeSelectingItemsTitleStyle("Select request type"),
		Options:     []string{BundleRequestType, IntegrationRequestType},
		FilterFunc:  func(s string) string { return s },
		DisplayFunc: func(s string) string { return s },
		IsEqual:     func(s string, s2 string) bool { return s == s2 },
		PostMessage: func(s []string) string {
			return styles.AfterSelectingItemsTitleStyle("Selected request type", []string{s[0]})
		},
	}

	selectedRequestTypes, err := listselect2.LaunchSelector(requestTypeInput)
	if err != nil {
		return "", err
	}

	return selectedRequestTypes[0], nil
}
