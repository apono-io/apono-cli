package assist

import (
	"fmt"
	"io"

	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/utils"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	titleWidthPadding = 4
	minTitleWidth     = 10
)

type convItem struct {
	conv clientapi.AssistantConversationClientModel
}

func (i convItem) FilterValue() string {
	return i.conv.Title
}

func (i convItem) Title() string {
	return i.conv.Title
}

func (i convItem) Description() string {
	return utils.FormatTimeAgo(i.conv.CreatedDate)
}

type convItemDelegate struct{}

func (d convItemDelegate) Height() int  { return 2 }
func (d convItemDelegate) Spacing() int { return 1 }
func (d convItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d convItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(convItem)
	if !ok {
		return
	}

	title := item.Title()
	timeAgo := item.Description()
	isSelected := index == m.Index()

	maxTitleWidth := m.Width() - titleWidthPadding
	if maxTitleWidth < minTitleWidth {
		maxTitleWidth = minTitleWidth
	}

	displayTitle := title
	if len(displayTitle) > maxTitleWidth {
		displayTitle = displayTitle[:maxTitleWidth-3] + "..."
	}

	var line1 string
	if isSelected {
		line1 = convItemSelectedStyle.Render("  " + displayTitle)
	} else {
		line1 = convItemStyle.Render("  " + displayTitle)
	}

	var line2 string
	if isSelected {
		line2 = convMetadataSelectedStyle.Render("  " + timeAgo)
	} else {
		line2 = convMetadataStyle.Render("  " + timeAgo)
	}

	_, _ = fmt.Fprintln(w, line1)
	_, _ = fmt.Fprint(w, line2)
}

func conversationsToItems(conversations []clientapi.AssistantConversationClientModel) []list.Item {
	items := make([]list.Item, len(conversations))
	for i, conv := range conversations {
		items[i] = convItem{conv: conv}
	}
	return items
}

func getSelectedConversation(l list.Model) *clientapi.AssistantConversationClientModel {
	item := l.SelectedItem()
	if item == nil {
		return nil
	}
	ci, ok := item.(convItem)
	if !ok {
		return nil
	}
	return &ci.conv
}

func newConvList(width, height int) list.Model {
	delegate := convItemDelegate{}
	l := list.New([]list.Item{}, delegate, width, height)

	l.Title = "Resume"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15"))
	l.Styles.PaginationStyle = lipgloss.NewStyle().
		Foreground(mutedColor)

	return l
}
