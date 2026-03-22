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
	noItemsTextColor   = color.Gray

	prefixNotice = "notice"
	prefixNew    = "NEW"
)

var (
	beforeSelectIcon = prefixIconColor.Sprint("?")
	afterSelectIcon  = prefixIconColor.Sprint("✓")
	noticeMsgPrefix  = color.Bold.Sprintf("[") + color.LightBlue.Sprintf(prefixNotice) + color.Bold.Sprintf("]")
	newMsgPrefix     = color.Bold.Sprintf("[") + color.Green.Sprintf(prefixNew) + color.Bold.Sprintf("]")
)

func GetNoticeMessagePrefix() string {
	return noticeMsgPrefix
}

func GetCustomMessagePrefix(prefix string) string {
	switch strings.ToUpper(prefix) {
	case prefixNew:
		return newMsgPrefix
	default:
		return noticeMsgPrefix
	}
}

func BeforeSelectingItemsTitleStyle(name string, optional bool) string {
	var optionalText string
	if optional {
		optionalText = optionalTextColor.Sprint(" (optional)")
	}

	return fmt.Sprintf("%s %s%s:", beforeSelectIcon, name, optionalText)
}

func AfterSelectingItemsTitleStyle(name string, items []string) string {
	itemsText := joinNames(items)
	if itemsText == "" {
		itemsText = noItemsTextColor.Sprint("(empty)")
	}

	return fmt.Sprintf("%s %s: %s", afterSelectIcon, name, itemsText)
}

func joinNames(names []string) string {
	var coloredNames []string
	for _, name := range names {
		if name != "" {
			coloredNames = append(coloredNames, selectedItemsColor.Sprint(name))
		}
	}

	return strings.Join(coloredNames, ", ")
}
