package list_select

import "github.com/charmbracelet/bubbles/list"

type SelectInput[T any] struct {
	Title             string
	Options           []T
	MultipleSelection bool
	PostMessage       func([]T) string
	FilterFunc        func(T) string
	DisplayFunc       func(T) string
	IsEqual           func(T, T) bool
	ShowHelp          bool
	EnableFilter      bool
	ShowItemCount     bool
}

type selectItem[T any] struct {
	data     T
	selected bool
	input    SelectInput[T]
}

type model[T any] struct {
	list       list.Model
	aborting   bool
	submitting bool
}
