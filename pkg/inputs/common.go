package inputs

import (
	"fmt"
	"strings"
)

const (
	submitKey            = "enter"
	multiSelectChoiceKey = " "
	AbortKey             = "ctrl+c"
	justificationQuitKey = "esc"
)

func hoveredItemRender(s ...string) string {
	return selectedItemStyle.Render("> " + strings.Join(s, " "))
}

func multiSelectItemRender(item string, selected bool) string {
	if selected {
		return fmt.Sprintf("[*] %s", item)
	} else {
		return fmt.Sprintf("[ ] %s", item)
	}
}
