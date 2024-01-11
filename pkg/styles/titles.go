package styles

import (
	"fmt"
	"strings"

	"github.com/gookit/color"
)

const (
	prefixIconColor    = color.Green
	selectedItemsColor = color.Cyan
)

var (
	beforeSelectIcon = prefixIconColor.Sprint("?")
	afterSelectIcon  = prefixIconColor.Sprint("✓")
)

func BeforeSelectingItemsTitleStyle(name string) string {
	return fmt.Sprintf("%s %s:", beforeSelectIcon, name)
}

func AfterSelectingItemsTitleStyle(name string, items []string) string {
	return fmt.Sprintf("%s %s: %s", afterSelectIcon, name, joinNames(items))
}

func joinNames(names []string) string {
	var coloredNames []string
	for _, name := range names {
		coloredNames = append(coloredNames, selectedItemsColor.Sprint(name))
	}

	return strings.Join(coloredNames, ", ")
}
