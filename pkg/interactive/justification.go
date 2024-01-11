package interactive

import (
	textinput "github.com/apono-io/apono-cli/pkg/interactive/inputs/text_input"
	"github.com/apono-io/apono-cli/pkg/styles"
)

func RunJustificationInput() (string, error) {
	justificationInput := textinput.TextInput{
		Title:       styles.BeforeSelectingItemsTitleStyle("Enter justification"),
		Placeholder: "Justification",
		PostMessage: func(s string) string {
			return styles.AfterSelectingItemsTitleStyle("Justification", []string{s})
		},
	}

	justification, err := textinput.LaunchTextInput(justificationInput)
	if err != nil {
		return "", err
	}

	return justification, nil
}
