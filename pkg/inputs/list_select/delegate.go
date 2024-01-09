package listselect

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type selectItemDelegate[T any] struct{}

func (i selectItem[T]) FilterValue() string { return i.input.FilterFunc(i.data) }

func (d selectItemDelegate[T]) Height() int { return 1 }

func (d selectItemDelegate[T]) Spacing() int { return 0 }

func (d selectItemDelegate[T]) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d selectItemDelegate[T]) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(selectItem[T])
	if !ok {
		return
	}

	var str string
	if item.input.MultipleSelection {
		str = multiSelectItemRender(item.input.DisplayFunc(item.data), item.selected)
	} else {
		str = item.input.DisplayFunc(item.data)
	}

	fn := itemStyle.Render
	if index == m.Index() {
		fn = hoveredItemRender
	}

	fmt.Fprint(w, fn(str))
}
