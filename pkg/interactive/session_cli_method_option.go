package interactive

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
			Label: "Run command",
		},
		{
			ID:    PrintOption,
			Label: "Print command",
		},
	}

	requestTypeInput := listselect.SelectInput{
		Title:     "Select how to use access command",
		PostTitle: "Selected option",
		Options:   options,
	}

	selectedItems, err := listselect.LaunchSelector(requestTypeInput)
	if err != nil {
		return "", err
	}

	return selectedItems[0].ID, nil
}
