package inputs

import (
	"fmt"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"io"
)

type resourceTypeItem clientapi.ResourceTypeClientModel

type resourceTypeItemDelegate struct{}

type resourceTypeSelectorModel struct {
	list     list.Model
	choice   *resourceTypeItem
	quitting bool
}

func (i resourceTypeItem) FilterValue() string { return i.Name + i.DisplayPath }

func (d resourceTypeItemDelegate) Height() int                             { return 1 }
func (d resourceTypeItemDelegate) Spacing() int                            { return 0 }
func (d resourceTypeItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d resourceTypeItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(resourceTypeItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%s", item.Name)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = hoveredItemRender
	}

	fmt.Fprint(w, fn(str))
}

func (m resourceTypeSelectorModel) Init() tea.Cmd { return nil }

func (m resourceTypeSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case AbortKey:
			m.quitting = true
			return m, tea.Quit

		case submitKey:
			item, ok := m.list.SelectedItem().(resourceTypeItem)
			if ok {
				m.choice = &item
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m resourceTypeSelectorModel) View() string {
	if m.choice != nil {
		return ""
	}
	if m.quitting {
		return abortingInputTitle
	}

	return m.list.View()
}

func LaunchResourceTypeSelector(options []clientapi.ResourceTypeClientModel) (*clientapi.ResourceTypeClientModel, error) {
	var items []list.Item
	for _, resourceType := range options {
		items = append(items, resourceTypeItem(resourceType))
	}

	l := list.New(items, resourceTypeItemDelegate{}, defaultListWidth, defaultListHeight)

	l.Title = getResourceTypeTitle("")
	l.AdditionalShortHelpKeys = singleSelectAdditionalHelpKeys
	l.AdditionalFullHelpKeys = singleSelectAdditionalHelpKeys
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	initModel := resourceTypeSelectorModel{list: l}
	result, err := tea.NewProgram(initModel).Run()
	if err != nil {
		return nil, err
	}

	resultModel := result.(resourceTypeSelectorModel)
	if resultModel.choice == nil {
		return nil, fmt.Errorf("no resource type selected")
	}

	fmt.Println(getResourceTypeTitle(resultModel.choice.Name))

	return (*clientapi.ResourceTypeClientModel)(resultModel.choice), nil
}
