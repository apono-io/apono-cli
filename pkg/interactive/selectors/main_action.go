package selectors

import listselect "github.com/apono-io/apono-cli/pkg/interactive/inputs/list_select"

const (
	RequestAccessOption = "request_access"
	ConnectOption       = "connect"
)

func RunMainActionSelector() (string, error) {
	options := []listselect.SelectOption{
		{
			ID:    RequestAccessOption,
			Label: "Request new access",
		},
		{
			ID:    ConnectOption,
			Label: "Connect to a resource",
		},
	}

	requestTypeInput := listselect.SelectInput{
		Title:     "What do you want to do?",
		PostTitle: "Selected action",
		Options:   options,
	}

	selectedItems, err := listselect.LaunchSelector(requestTypeInput)
	if err != nil {
		return "", err
	}

	return selectedItems[0].ID, nil
}
