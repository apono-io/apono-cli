package interactive

import (
	listselect "github.com/apono-io/apono-cli/pkg/interactive/inputs/list_select"
	"github.com/apono-io/apono-cli/pkg/styles"
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
		Title:   styles.BeforeSelectingItemsTitleStyle("Select how to use access command"),
		Options: options,
		PostMessage: func(s []listselect.SelectOption) string {
			return styles.AfterSelectingItemsTitleStyle("Selected option", []string{s[0].Label})
		},
	}

	selectedItems, err := listselect.LaunchSelector(requestTypeInput)
	if err != nil {
		return "", err
	}

	return selectedItems[0].ID, nil
}
