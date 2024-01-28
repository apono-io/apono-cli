package selectors

import (
	listselect "github.com/apono-io/apono-cli/pkg/interactive/inputs/list_select"
)

const (
	ExecuteOption = "execute"
	PrintOption   = "print"
)

func RunSessionCliMethodOptionSelector() (string, error) {
	options := []listselect.SelectOption{
		{
			ID:    ExecuteOption,
			Label: "Connect",
		},
		{
			ID:    PrintOption,
			Label: "Instructions",
		},
	}

	requestTypeInput := listselect.SelectInput{
		Title:     "Select how to use access",
		PostTitle: "Selected option",
		Options:   options,
	}

	selectedItems, err := listselect.LaunchSelector(requestTypeInput)
	if err != nil {
		return "", err
	}

	return selectedItems[0].ID, nil
}
