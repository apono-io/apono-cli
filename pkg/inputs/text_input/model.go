package textinput

type TextInput struct {
	Title       string
	Placeholder string
	PostMessage func(string) string
}
