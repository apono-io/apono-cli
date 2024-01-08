package inputs

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type errMsg error

type justificationModel struct {
	textInput  textinput.Model
	err        error
	submitting bool
	quitting   bool
}

func initialModel() justificationModel {
	ti := textinput.New()
	ti.Placeholder = "Need Access"
	ti.Focus()
	//ti.CharLimit = 156
	ti.Width = 200

	return justificationModel{
		textInput: ti,
		err:       nil,
	}
}

func (m justificationModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m justificationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case AbortKey, justificationQuitKey:
			m.quitting = true
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

func (m justificationModel) View() string {
	if m.quitting {
		return abortingInputTitle
	}
	if m.submitting {
		return ""
	}

	return fmt.Sprintf("%s \n\n%s\n\n%s",
		getJustificationTitle(""),
		m.textInput.View(),
		"(esc/ctrl+c to abort)",
	)
}

func LaunchJustificationInput() (string, error) {
	result, err := tea.NewProgram(initialModel()).Run()
	if err != nil {
		return "", err
	}

	resultModel := result.(justificationModel)

	if resultModel.quitting {
		return "", nil
	}

	justification := resultModel.textInput.Value()
	fmt.Println(getJustificationTitle(justification))

	return justification, nil
}
