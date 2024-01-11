package listselect

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

const (
	defaultListHeight = 20
	defaultListWidth  = 1
)

var (
	defaultTitleStyle        = lipgloss.NewStyle().Margin(0, 0, 0, 0)
	defaultItemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	defaultSelectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("6"))
	defaultNoSelectionStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#9e413c"))
	defaultPaginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	defaultHelpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
)

func hoveredItemRender(s ...string) string {
	return defaultSelectedItemStyle.Render("> " + strings.Join(s, " "))
}

func multiSelectItemRender(item string, selected bool) string {
	if selected {
		return fmt.Sprintf("[*] %s", item)
	} else {
		return fmt.Sprintf("[ ] %s", item)
	}
}
