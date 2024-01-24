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

type selectItem struct {
	data     SelectOption
	selected bool
	input    SelectInput
}

type model struct {
	list       list.Model
	aborting   bool
	submitting bool
}
