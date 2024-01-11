package interactive

import (
	textinput2 "github.com/apono-io/apono-cli/pkg/interactive/inputs/text_input"
	"github.com/apono-io/apono-cli/pkg/styles"
)

func RunJustificationInput() (string, error) {
	justificationInput := textinput2.TextInput{
		Title:       styles.BeforeSelectingItemsTitleStyle("Enter justification"),
		Placeholder: "Justification",
		PostMessage: func(s string) string {
			return styles.AfterSelectingItemsTitleStyle("Justification", []string{s})
		},
	}

	justification, err := textinput2.LaunchTextInput(justificationInput)
	if err != nil {
		return "", err
	}

	return justification, nil
}
