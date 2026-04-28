package listselect

import "github.com/charmbracelet/bubbles/list"

type SelectOption struct {
	ID    string
	Label string
}

type SelectInput struct {
	Title                string
	PostTitle            string
	Options              []SelectOption
	MultipleSelection    bool
	ShowHelp             bool
	EnableFilter         bool
	ShowItemCount        bool
	AutoSelectSingleItem bool
}

// selectionState is shared by pointer across all copies of selectItem
// (including filtered copies) so toggling selection doesn't require
// modifying list items, which would reset the filter.
type selectionState struct {
	selected map[string]bool
}

type selectItem struct {
	data  SelectOption
	state *selectionState
	input SelectInput
}

func (i selectItem) isSelected() bool {
	return i.state.selected[i.data.ID]
}

type model struct {
	list       list.Model
	aborting   bool
	submitting bool
}
