package assist

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

const slashCommandNameWidth = 18

type slashItem struct {
	cmd SlashCommand
}

func (i slashItem) FilterValue() string { return i.cmd.Name }
func (i slashItem) Title() string       { return i.cmd.Name }
func (i slashItem) Description() string { return i.cmd.Description }

type slashItemDelegate struct{}

func (d slashItemDelegate) Height() int                             { return 1 }
func (d slashItemDelegate) Spacing() int                            { return 0 }
func (d slashItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d slashItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(slashItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	name := fmt.Sprintf("%-*s", slashCommandNameWidth, item.cmd.Name)

	var line string
	if isSelected {
		line = slashItemSelectedStyle.Render("  "+name) + slashDescStyle.Render(item.cmd.Description)
	} else {
		line = slashItemStyle.Render("  "+name) + slashDescStyle.Render(item.cmd.Description)
	}

	_, _ = fmt.Fprint(w, line)
}

func newSlashList(width, height int) list.Model {
	delegate := slashItemDelegate{}
	l := list.New([]list.Item{}, delegate, width, height)

	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)

	return l
}

func slashCommandsToItems(commands []SlashCommand) []list.Item {
	items := make([]list.Item, len(commands))
	for i, cmd := range commands {
		items[i] = slashItem{cmd: cmd}
	}
	return items
}
