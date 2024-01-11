package listselect

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type selectItemDelegate struct{}

func (i selectItem) FilterValue() string { return i.data.Filter }

func (d selectItemDelegate) Height() int { return 1 }

func (d selectItemDelegate) Spacing() int { return 0 }

func (d selectItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d selectItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(selectItem)
	if !ok {
		return
	}

	var str string
	if item.input.MultipleSelection {
		str = multiSelectItemRender(item.data.Label, item.selected)
	} else {
		str = item.data.Label
	}

	fn := defaultItemStyle.Render
	if index == m.Index() {
		fn = hoveredItemRender
	}

	fmt.Fprint(w, fn(str))
}
