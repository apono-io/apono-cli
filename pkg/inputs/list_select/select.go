package list_select

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	abortingText = "Aborting..."
	noSelectText = "No items selected"
)

func (m model[T]) Init() tea.Cmd { return nil }

func (m model[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case abortKey, quitKey:
			m.aborting = true
			return m, tea.Quit

		case submitKey:
			m.submitting = true
			item, ok := m.list.SelectedItem().(selectItem[T])
			if ok {
				if !item.input.MultipleSelection {
					m.list.SetItems(handleItemSelection[T](m.list.Items(), item))
				} else {
					if getNumberOfSelectedItems[T](m.list.Items()) == 0 {
						m.submitting = false
						m.list.NewStatusMessage(noSelectionStyle.Render(noSelectText))
						return m, nil
					}
				}
			}
			return m, tea.Quit

		case multiSelectChoiceKey:
			m.list.NewStatusMessage("")
			item, ok := m.list.SelectedItem().(selectItem[T])
			if ok {
				if item.input.MultipleSelection {
					m.list.SetItems(handleItemSelection[T](m.list.Items(), item))
				}
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func handleItemSelection[T any](items []list.Item, selectedItem selectItem[T]) []list.Item {
	for index, item := range items {
		currentItem := item.(selectItem[T])
		if currentItem.input.IsEqual(currentItem.data, selectedItem.data) {
			currentItem.selected = !currentItem.selected
			items[index] = currentItem
		}
	}

	return items
}

func getNumberOfSelectedItems[T any](items []list.Item) int {
	var count int
	for _, item := range items {
		if item.(selectItem[T]).selected {
			count++
		}
	}

	return count
}

func (m model[T]) View() string {
	if m.submitting {
		return ""
	}
	if m.aborting {
		return titleStyle.Render(abortingText)
	}

	return m.list.View()
}

func LaunchSelector[T any](inputModel SelectInput[T]) ([]T, error) {
	var items []list.Item
	for _, option := range inputModel.Options {
		items = append(items, selectItem[T]{data: option, input: inputModel})
	}

	l := list.New(items, selectItemDelegate[T]{}, defaultListWidth, defaultListHeight)

	if inputModel.MultipleSelection {
		l.AdditionalShortHelpKeys = multiSelectAdditionalHelpKeys
		l.AdditionalFullHelpKeys = multiSelectAdditionalHelpKeys
	} else {
		l.AdditionalShortHelpKeys = singleSelectAdditionalHelpKeys
		l.AdditionalFullHelpKeys = singleSelectAdditionalHelpKeys
	}

	l.SetShowHelp(inputModel.ShowHelp)
	l.SetFilteringEnabled(inputModel.EnableFilter)
	l.SetShowStatusBar(inputModel.ShowItemCount)

	l.Title = inputModel.Title
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	initModel := model[T]{list: l}
	result, err := tea.NewProgram(initModel).Run()
	if err != nil {
		return nil, err
	}

	resultModel := result.(model[T])
	if resultModel.aborting {
		return nil, fmt.Errorf("aborted by user")
	}

	var selectedItems []T
	for _, item := range resultModel.list.Items() {
		if item.(selectItem[T]).selected {
			selectedItems = append(selectedItems, item.(selectItem[T]).data)
		}
	}

	if len(selectedItems) == 0 {
		return nil, fmt.Errorf("no items selected")
	}

	if inputModel.PostMessage != nil {
		fmt.Println(inputModel.PostMessage(selectedItems))
	}

	return selectedItems, nil
}
