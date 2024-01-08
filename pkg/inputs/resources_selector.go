package inputs

import (
	"fmt"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"io"
)

type resourceItem struct {
	resource clientapi.ResourceClientModel
	selected bool
}

type resourceItemDelegate struct{}

type resourcesSelectorModel struct {
	list       list.Model
	submitting bool
	quitting   bool
}

func (i resourceItem) FilterValue() string { return i.resource.Path }

func (d resourceItemDelegate) Height() int                             { return 1 }
func (d resourceItemDelegate) Spacing() int                            { return 0 }
func (d resourceItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d resourceItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(resourceItem)
	if !ok {
		return
	}

	str := multiSelectItemRender(item.resource.Path, item.selected)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = hoveredItemRender
	}

	fmt.Fprint(w, fn(str))
}

func (m resourcesSelectorModel) Init() tea.Cmd { return nil }

func (m resourcesSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case AbortKey:
			m.quitting = true
			return m, tea.Quit

		case multiSelectChoiceKey:
			i, ok := m.list.SelectedItem().(resourceItem)
			if ok {
				m.list.SetItems(handleResourceItemSelection(m.list.Items(), i))
			}
			return m, nil

		case submitKey:
			m.submitting = true
			return m, tea.Quit
		}

	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m resourcesSelectorModel) View() string {
	if m.submitting {
		return ""
	}
	if m.quitting {
		return abortingInputTitle
	}

	return m.list.View()
}

func handleResourceItemSelection(items []list.Item, selectedItem resourceItem) []list.Item {
	for index, item := range items {
		currentResourceItem := item.(resourceItem)
		if currentResourceItem.resource.Id == selectedItem.resource.Id {
			items[index] = resourceItem{resource: currentResourceItem.resource, selected: !currentResourceItem.selected}
		}
	}

	return items
}

func LaunchResourcesSelector(
	resources []clientapi.ResourceClientModel,
) ([]clientapi.ResourceClientModel, error) {

	var items []list.Item
	for _, resource := range resources {
		items = append(items, resourceItem{resource: resource})
	}

	l := list.New(items, resourceItemDelegate{}, defaultListWidth, defaultListHeight)

	l.Title = getResourcesTitle([]string{})
	l.AdditionalShortHelpKeys = multiSelectAdditionalHelpKeys
	l.AdditionalFullHelpKeys = multiSelectAdditionalHelpKeys
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	m := resourcesSelectorModel{list: l}

	result, err := tea.NewProgram(m).Run()
	if err != nil {
		return nil, err
	}

	resultModel := result.(resourcesSelectorModel)
	var resultResources []clientapi.ResourceClientModel
	var resultResourcesNames []string
	for _, item := range resultModel.list.Items() {
		if item.(resourceItem).selected {
			resultResources = append(resultResources, item.(resourceItem).resource)
			resultResourcesNames = append(resultResourcesNames, item.(resourceItem).resource.Path)
		}
	}

	if resultResourcesNames == nil {
		return nil, fmt.Errorf("no resources selected")
	}

	fmt.Println(getResourcesTitle(resultResourcesNames))

	return resultResources, nil
}
