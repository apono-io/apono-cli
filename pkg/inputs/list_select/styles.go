package list_select

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
	titleStyle        = lipgloss.NewStyle().Margin(0, 0, 0, 0)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("6"))
	noSelectionStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#9e413c"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
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
