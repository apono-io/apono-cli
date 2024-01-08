package inputs

import (
	"fmt"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"io"
)

type integrationItem clientapi.IntegrationClientModel

type integrationItemDelegate struct{}

type IntegrationSelectorModel struct {
	list     list.Model
	choice   *integrationItem
	quitting bool
}

func (i integrationItem) FilterValue() string { return i.Name + i.TypeDisplayName }

func (d integrationItemDelegate) Height() int                             { return 1 }
func (d integrationItemDelegate) Spacing() int                            { return 0 }
func (d integrationItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d integrationItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(integrationItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%s/%s", item.TypeDisplayName, item.Name)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = hoveredItemRender
	}

	fmt.Fprint(w, fn(str))
}

func (m IntegrationSelectorModel) Init() tea.Cmd { return nil }

func (m IntegrationSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			item, ok := m.list.SelectedItem().(integrationItem)
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

func (m IntegrationSelectorModel) View() string {
	if m.choice != nil {
		return ""
	}
	if m.quitting {
		return abortingInputTitle
	}

	return m.list.View()
}

func LaunchIntegrationSelector(options []clientapi.IntegrationClientModel) (*clientapi.IntegrationClientModel, error) {
	var items []list.Item
	for _, integration := range options {
		items = append(items, integrationItem(integration))
	}

	l := list.New(items, integrationItemDelegate{}, defaultListWidth, defaultListHeight)

	l.Title = getIntegrationTitle("")
	l.AdditionalShortHelpKeys = singleSelectAdditionalHelpKeys
	l.AdditionalFullHelpKeys = singleSelectAdditionalHelpKeys
	//l.Help
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	initModel := IntegrationSelectorModel{list: l}
	result, err := tea.NewProgram(initModel).Run()
	if err != nil {
		return nil, err
	}

	resultModel := result.(IntegrationSelectorModel)
	if resultModel.choice == nil {
		return nil, fmt.Errorf("no integration selected")
	}

	fmt.Println(getIntegrationTitle(resultModel.choice.Name))

	return (*clientapi.IntegrationClientModel)(resultModel.choice), nil
}
