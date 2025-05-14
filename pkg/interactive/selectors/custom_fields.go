package selectors

import (
	"fmt"

	"github.com/apono-io/apono-cli/pkg/clientapi"
	listselect "github.com/apono-io/apono-cli/pkg/interactive/inputs/list_select"
	textinput "github.com/apono-io/apono-cli/pkg/interactive/inputs/text_input"
)

const (
	fieldTypeText   = "TEXT"
	fieldTypeSelect = "SELECT"
)

var emptySelectOptionField = &listselect.SelectOption{
	ID:    "empty",
	Label: "-- Skip this field --",
}

func RunCustomFieldsInputs(customFields []clientapi.RequestCustomFieldModel) (map[string]string, error) {
	result := make(map[string]string)

	for _, field := range customFields {
		switch field.Type {
		case fieldTypeText:
			value, err := runTextFieldInput(field)
			if err != nil {
				return nil, err
			}
			if value != "" {
				result[field.Id] = value
			}
		case fieldTypeSelect:
			value, err := runSelectFieldInput(field)
			if err != nil {
				return nil, err
			}
			if value != "" {
				result[field.Id] = value
			}
		}
	}

	return result, nil
}

func runTextFieldInput(field clientapi.RequestCustomFieldModel) (string, error) {
	defaultValue := ""
	if field.Default.IsSet() {
		defaultValue = *field.Default.Get()
	}

	textInput := textinput.TextInput{
		Title:        fmt.Sprintf("Enter %s", field.Label),
		PostTitle:    field.Label,
		Placeholder:  field.Placeholder,
		Optional:     !field.Required,
		InitialValue: defaultValue,
	}

	value, err := textinput.LaunchTextInput(textInput)
	if err != nil {
		return "", err
	}

	return value, nil
}

func runSelectFieldInput(field clientapi.RequestCustomFieldModel) (string, error) {
	defaultValueKey := ""
	if field.Default.IsSet() {
		defaultValueKey = *field.Default.Get()
	}

	var orderedOptions []listselect.SelectOption

	for _, value := range field.Values {
		if value.Key == defaultValueKey {
			orderedOptions = append(orderedOptions, listselect.SelectOption{
				ID:    value.Key,
				Label: fmt.Sprintf("%s (default)", value.Value),
			})
			break
		}
	}

	for _, value := range field.Values {
		if value.Key != defaultValueKey {
			orderedOptions = append(orderedOptions, listselect.SelectOption{
				ID:    value.Key,
				Label: value.Value,
			})
		}
	}

	if !field.Required {
		orderedOptions = append(orderedOptions, *emptySelectOptionField)
	}

	selectInput := listselect.SelectInput{
		Title:             fmt.Sprintf("Select %s", field.Label),
		PostTitle:         field.Label,
		Options:           orderedOptions,
		MultipleSelection: false,
		ShowHelp:          true,
		EnableFilter:      true,
		ShowItemCount:     true,
	}

	selectedItems, err := listselect.LaunchSelector(selectInput)
	if err != nil {
		return "", err
	}

	selectedKey := selectedItems[0].ID
	if selectedKey == emptySelectOptionField.ID {
		if field.Required {
			return "", fmt.Errorf("required field '%s' must have a value", field.Label)
		}
		return "", nil
	}

	return selectedKey, nil
}
