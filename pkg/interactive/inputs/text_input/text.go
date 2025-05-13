package textinput

import (
	"fmt"

	"github.com/apono-io/apono-cli/pkg/styles"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	inputWidth     = 200
	abortingText   = "Aborting..."
	noTextInputMsg = "Input is required"
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
			if !m.optional && m.textInput.Value() == "" {
				m.statusMsg = defaultNoInputStyle.Render(noTextInputMsg)
				return m, nil
			} else {
				m.submitting = true
				return m, tea.Quit
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

func initialModel(title string, placeholder string, optional bool, initialValue string) model {
	ti := textinput.New()
	ti.Focus()
	ti.Placeholder = placeholder
	ti.Width = inputWidth

	if initialValue != "" {
		ti.SetValue(initialValue)
		ti.CursorEnd()
	}

	return model{
		textInput: ti,
		title:     styles.BeforeSelectingItemsTitleStyle(title, optional),
		err:       nil,
		optional:  optional,
	}
}

func LaunchTextInput(input TextInput) (string, error) {
	result, err := tea.NewProgram(initialModel(
		input.Title,
		input.Placeholder,
		input.Optional,
		input.InitialValue,
	)).Run()
	if err != nil {
		return "", err
	}

	resultModel := result.(model)
	if resultModel.aborting {
		return "", fmt.Errorf("aborted by user")
	}

	resultText := resultModel.textInput.Value()
	if resultText == "" && !input.Optional {
		return "", fmt.Errorf("no input provided")
	}

	if input.PostTitle != "" {
		fmt.Println(styles.AfterSelectingItemsTitleStyle(input.PostTitle, []string{resultText}))
	}

	return resultText, nil
}
