package interactive

import (
	textinput "github.com/apono-io/apono-cli/pkg/interactive/inputs/text_input"
)

func RunJustificationInput() (string, error) {
	justificationInput := textinput.TextInput{
		Title:       "Enter justification",
		PostTitle:   "Justification",
		Placeholder: "Justification",
	}

	justification, err := textinput.LaunchTextInput(justificationInput)
	if err != nil {
		return "", err
	}

	return justification, nil
}
