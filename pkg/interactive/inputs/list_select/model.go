package listselect

import "github.com/charmbracelet/bubbles/list"

type SelectOption struct {
	ID    string
	Label string
}

type SelectInput struct {
	Title             string
	Options           []SelectOption
	MultipleSelection bool
	PostMessage       func([]SelectOption) string
	ShowHelp          bool
	EnableFilter      bool
	ShowItemCount     bool
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
