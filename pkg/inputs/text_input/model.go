package text_input

type TextInput struct {
	Title       string
	Placeholder string
	PostMessage func(string) string
}
