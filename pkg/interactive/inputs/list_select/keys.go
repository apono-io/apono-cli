package listselect

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
)

const (
	submitKey            = "enter"
	multiSelectChoiceKey = " "
	abortKey             = "ctrl+c"
	quitKey              = "esc"
)

var (
	abortKeyBinding = key.NewBinding(
		key.WithKeys(quitKey, abortKey),
		key.WithHelp(fmt.Sprintf("%s/%s", quitKey, abortKey), "abort"),
	)
	submitKeyBinding = key.NewBinding(
		key.WithKeys(submitKey),
		key.WithHelp(submitKey, "submit"),
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
