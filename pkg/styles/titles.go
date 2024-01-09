package styles

import (
	"fmt"

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
	var result string
	for i, name := range names {
		if i != len(names)-1 {
			result += selectedItemsColor.Sprint(name) + ", "
		} else {
			result += selectedItemsColor.Sprint(name)
		}
	}

	return result
}
