package selectors

import (
	textinput "github.com/apono-io/apono-cli/pkg/interactive/inputs/text_input"
)

func RunJustificationInput(optional bool) (string, error) {
	justificationInput := textinput.TextInput{
		Title:       "Enter Justification",
		PostTitle:   "Justification",
		Placeholder: "Justification",
		Optional:    optional,
	}

	justification, err := textinput.LaunchTextInput(justificationInput)
	if err != nil {
		return "", err
	}

	return justification, nil
}
