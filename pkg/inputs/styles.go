package inputs

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/gookit/color"
)

const (
	defaultListHeight = 20
	defaultListWidth  = 1
	prefixIconColor   = color.Green
	chosenItemsColor  = color.Cyan
)

var (
	titleStyle        = lipgloss.NewStyle().Margin(0, 0, 0, 0)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("6"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = titleStyle
)
