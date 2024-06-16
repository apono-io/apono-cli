package styles

import (
	"fmt"
	"strings"

	"github.com/gookit/color"
)

const (
	prefixIconColor    = color.Green
	selectedItemsColor = color.Cyan
	optionalTextColor  = color.Gray
)

var (
	beforeSelectIcon = prefixIconColor.Sprint("?")
	afterSelectIcon  = prefixIconColor.Sprint("✓")
	NoticeMsgPrefix  = color.Bold.Sprintf("[") + color.LightBlue.Sprintf("notice") + color.Bold.Sprintf("]")
)

func BeforeSelectingItemsTitleStyle(name string, optional bool) string {
	var optionalText string
	if optional {
		optionalText = optionalTextColor.Sprint(" (optional)")
	}

	return fmt.Sprintf("%s %s%s:", beforeSelectIcon, name, optionalText)
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
