package listselect

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	abortingText = "Aborting..."
	noSelectText = "No items selected"
)

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case abortKey:
			m.aborting = true
			return m, tea.Quit
		case quitKey:
			if m.list.FilterState() != list.Filtering {
				m.aborting = true
				return m, tea.Quit
			}
		case submitKey:
			if m.list.FilterState() != list.Filtering {
				m.submitting = true
				item, ok := m.list.SelectedItem().(selectItem)
				if ok {
					if item.input.MultipleSelection {
						if getNumberOfSelectedItems(m.list.Items()) == 0 {
							m.submitting = false
							m.list.NewStatusMessage(defaultNoSelectionStyle.Render(noSelectText))
							return m, nil
						}
					} else {
						m.list.SetItems(handleItemSelection(m.list.Items(), item))
					}
				}

				return m, tea.Quit
			}

		case multiSelectChoiceKey:
			if m.list.FilterState() != list.Filtering {
				m.list.NewStatusMessage("")
				item, ok := m.list.SelectedItem().(selectItem)
				if ok {
					if item.input.MultipleSelection {
						m.list.SetItems(handleItemSelection(m.list.Items(), item))
					}
				}
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func handleItemSelection(items []list.Item, selectedItem selectItem) []list.Item {
	for index, item := range items {
		currentItem := item.(selectItem)
		if currentItem.data.ID == selectedItem.data.ID {
			currentItem.selected = !currentItem.selected
			items[index] = currentItem
		}
	}

	return items
}

func getNumberOfSelectedItems(items []list.Item) int {
	var count int
	for _, item := range items {
		if item.(selectItem).selected {
			count++
		}
	}

	return count
}

func (m model) View() string {
	if m.submitting {
		return ""
	}
	if m.aborting {
		return defaultTitleStyle.Render(abortingText)
	}

	return m.list.View()
}

func LaunchSelector(inputModel SelectInput) ([]SelectOption, error) {
	var items []list.Item
	for _, option := range inputModel.Options {
		items = append(items, selectItem{data: option, input: inputModel})
	}

	l := list.New(items, selectItemDelegate{}, defaultListWidth, defaultListHeight)

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
	l.Styles.Title = defaultTitleStyle
	l.Styles.PaginationStyle = defaultPaginationStyle
	l.Styles.HelpStyle = defaultHelpStyle

	initModel := model{list: l}
	result, err := tea.NewProgram(initModel).Run()
	if err != nil {
		return nil, err
	}

	resultModel := result.(model)
	if resultModel.aborting {
		return nil, fmt.Errorf("aborted by user")
	}

	var selectedItems []SelectOption
	for _, item := range resultModel.list.Items() {
		if item.(selectItem).selected {
			selectedItems = append(selectedItems, item.(selectItem).data)
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
