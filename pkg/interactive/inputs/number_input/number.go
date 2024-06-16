package numberinput

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"

	"github.com/apono-io/apono-cli/pkg/styles"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	inputWidth     = 200
	abortingText   = "Aborting..."
	noTextInputMsg = "Input is required"
	notValidNumber = "Input is not a valid number"
)

var helpText = fmt.Sprintf("(%s/%s to abort or %s to submit)", abortKey, quitKey, submitKey)

type errMsg error

type model struct {
	textInput  textinput.Model
	title      string
	err        error
	submitting bool
	aborting   bool
	optional   bool
	statusMsg  string
	maxValue   *float64
	minValue   *float64
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case abortKey, quitKey:
			m.aborting = true
			return m, tea.Quit

		case submitKey:
			switch {
			case m.textInput.Value() == "":
				if m.optional {
					m.submitting = true
					return m, tea.Quit
				} else {
					m.statusMsg = defaultNoInputStyle.Render(noTextInputMsg)
					return m, nil
				}

			case !isValueNumber(m.textInput.Value()):
				m.statusMsg = defaultNoInputStyle.Render(notValidNumber)
				return m, nil
			default:
				value, _ := convertToFloat(m.textInput.Value())
				validationErr := validateValueInRange(value, m.maxValue, m.minValue)
				if validationErr != nil {
					m.statusMsg = defaultNoInputStyle.Render(validationErr.Error())
					return m, nil
				} else {
					m.submitting = true
					return m, tea.Quit
				}
			}
		}

	case errMsg:
		m.err = msg
		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.submitting {
		return ""
	}
	if m.aborting {
		return abortingText
	}

	return fmt.Sprintf("%s %s\n\n%s\n\n%s",
		m.title,
		m.statusMsg,
		m.textInput.View(),
		helpText,
	)
}

func initialModel(title string, placeholder string, optional bool, maxValue *float64, minValue *float64) model {
	ti := textinput.New()
	ti.Focus()
	ti.Placeholder = placeholder
	ti.Width = inputWidth

	return model{
		textInput: ti,
		title:     styles.BeforeSelectingItemsTitleStyle(title, optional),
		err:       nil,
		optional:  optional,
		maxValue:  maxValue,
		minValue:  minValue,
	}
}

func validateValueInRange(value float64, maxValue *float64, minValue *float64) error {
	if maxValue != nil && value > *maxValue {
		return fmt.Errorf("maximum allowed value is %.2f", *maxValue)
	}

	if minValue != nil && value <= *minValue {
		return fmt.Errorf("value should be greater than %.2f", *minValue)
	}

	return nil
}

func isValueNumber(input string) bool {
	_, err := convertToFloat(input)
	return err == nil
}

func convertToFloat(input string) (float64, error) {
	return strconv.ParseFloat(input, 64)
}

func LaunchNumberInput(input NumberInput) (*float64, error) {
	if (input.MaxValue != nil && input.MinValue != nil) && *input.MaxValue < *input.MinValue {
		return nil, fmt.Errorf("max value is less than min value")
	}

	result, err := tea.NewProgram(initialModel(input.Title, input.Placeholder, input.Optional, input.MaxValue, input.MinValue)).Run()
	if err != nil {
		return nil, err
	}

	resultModel := result.(model)
	if resultModel.aborting {
		return nil, fmt.Errorf("aborted by user")
	}

	var resultNumber *float64
	resultText := resultModel.textInput.Value()
	if resultText == "" {
		if !input.Optional {
			return nil, fmt.Errorf("no input provided")
		}

		resultNumber = nil
	} else {
		var convertNumber float64
		convertNumber, err = convertToFloat(resultText)
		if err != nil {
			return nil, fmt.Errorf("failed to convert input to number")
		}

		validationErr := validateValueInRange(convertNumber, input.MaxValue, input.MinValue)
		if validationErr != nil {
			return nil, validationErr
		}

		resultNumber = &convertNumber
	}

	if input.PostTitle != "" {
		fmt.Println(styles.AfterSelectingItemsTitleStyle(input.PostTitle, []string{resultText}))
	}

	return resultNumber, nil
}
