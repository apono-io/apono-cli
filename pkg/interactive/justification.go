package interactive

import (
	"github.com/apono-io/apono-cli/pkg/inputs/text_input"
	"github.com/apono-io/apono-cli/pkg/styles"
)

func RunJustificationInput() (string, error) {
	justificationInput := text_input.TextInput{
		Title:       styles.BeforeSelectingItemsTitleStyle("Enter justification"),
		Placeholder: "Justification",
		PostMessage: func(s string) string {
			return styles.AfterSelectingItemsTitleStyle("Justification", []string{s})
		},
	}

	justification, err := text_input.LaunchTextInput(justificationInput)
	if err != nil {
		return "", err
	}

	return justification, nil
}
