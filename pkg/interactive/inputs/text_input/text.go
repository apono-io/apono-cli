package textinput

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	inputWidth   = 200
	abortingText = "Aborting..."
)

var (
	helpText = fmt.Sprintf("(%s/%s to abort or %s to submit)", abortKey, quitKey, submitKey)
)

type errMsg error

type model struct {
	textInput  textinput.Model
	title      string
	err        error
	submitting bool
	aborting   bool
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
			m.submitting = true
			return m, tea.Quit
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

	return fmt.Sprintf("%s \n\n%s\n\n%s",
		m.title,
		m.textInput.View(),
		helpText,
	)
}

func initialModel(title string, placeholder string) model {
	ti := textinput.New()
	ti.Focus()
	ti.Placeholder = placeholder
	ti.Width = inputWidth

	return model{
		textInput: ti,
		title:     title,
		err:       nil,
	}
}

func LaunchTextInput(input TextInput) (string, error) {
	result, err := tea.NewProgram(initialModel(input.Title, input.Placeholder)).Run()
	if err != nil {
		return "", err
	}

	resultModel := result.(model)
	if resultModel.aborting {
		return "", fmt.Errorf("aborted by user")
	}

	justification := resultModel.textInput.Value()
	if input.PostMessage != nil {
		fmt.Println(input.PostMessage(justification))
	}

	return justification, nil
}
