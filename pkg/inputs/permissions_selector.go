package inputs

import (
	"fmt"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"io"
)

type permissionItem struct {
	permission clientapi.PermissionClientModel
	selected   bool
}

type permissionItemDelegate struct{}

type permissionSelectorModel struct {
	list                       list.Model
	multiplePermissionsAllowed bool
	submitting                 bool
	quitting                   bool
}

func (i permissionItem) FilterValue() string { return i.permission.Name }

func (d permissionItemDelegate) Height() int                             { return 1 }
func (d permissionItemDelegate) Spacing() int                            { return 0 }
func (d permissionItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d permissionItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(permissionItem)
	if !ok {
		return
	}

	str := multiSelectItemRender(item.permission.Name, item.selected)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = hoveredItemRender
	}

	fmt.Fprint(w, fn(str))
}

func (m permissionSelectorModel) Init() tea.Cmd { return nil }

func (m permissionSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			i, ok := m.list.SelectedItem().(permissionItem)
			if ok {
				m.list.SetItems(handlePermissionItemSelection(m.list.Items(), i))
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

func (m permissionSelectorModel) View() string {
	if m.submitting {
		return ""
	}
	if m.quitting {
		return abortingInputTitle
	}

	return m.list.View()
}

func handlePermissionItemSelection(items []list.Item, selectedItem permissionItem) []list.Item {
	for index, item := range items {
		currentResourceItem := item.(permissionItem)
		if currentResourceItem.permission.Id == selectedItem.permission.Id {
			items[index] = permissionItem{permission: currentResourceItem.permission, selected: !currentResourceItem.selected}
		}
	}

	return items
}

func LaunchPermissionsSelector(
	options []clientapi.PermissionClientModel,
	allowMultiplePermissions bool,
) ([]clientapi.PermissionClientModel, error) {
	var items []list.Item
	for _, permission := range options {
		items = append(items, permissionItem{permission: permission})
	}

	l := list.New(items, permissionItemDelegate{}, defaultListWidth, defaultListHeight)

	l.Title = getPermissionsTitle([]string{})
	l.AdditionalShortHelpKeys = multiSelectAdditionalHelpKeys
	l.AdditionalFullHelpKeys = multiSelectAdditionalHelpKeys
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	initModel := permissionSelectorModel{list: l, multiplePermissionsAllowed: allowMultiplePermissions}

	result, err := tea.NewProgram(initModel).Run()
	if err != nil {
		return nil, err
	}

	resultModel := result.(permissionSelectorModel)
	var resultPermissions []clientapi.PermissionClientModel
	var resultPermissionsNames []string
	for _, item := range resultModel.list.Items() {
		if item.(permissionItem).selected {
			resultPermissions = append(resultPermissions, item.(permissionItem).permission)
			resultPermissionsNames = append(resultPermissionsNames, item.(permissionItem).permission.Name)
		}
	}

	if resultPermissions == nil {
		return nil, fmt.Errorf("no permissions selected")
	}

	fmt.Println(getPermissionsTitle(resultPermissionsNames))

	return resultPermissions, nil
}
