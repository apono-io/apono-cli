package inputs

import "github.com/charmbracelet/bubbles/key"

var (
	abortKeyBinding = key.NewBinding(
		key.WithKeys("esc", "ctrl+c"),
		key.WithHelp("esc/ctrl+c", "abort"),
	)
	submitKeyBinding = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "submit"),
	)
	selectKeyBinding = key.NewBinding(
		key.WithKeys("space"),
		key.WithHelp("space", "select"),
	)
)

func singleSelectAdditionalHelpKeys() []key.Binding {
	return []key.Binding{abortKeyBinding, submitKeyBinding}
}

func multiSelectAdditionalHelpKeys() []key.Binding {
	return []key.Binding{abortKeyBinding, submitKeyBinding, selectKeyBinding}
}
