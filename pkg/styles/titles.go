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
)

var (
	beforeSelectIcon = prefixIconColor.Sprint("?")
	afterSelectIcon  = prefixIconColor.Sprint("✓")
	noticeMsgPrefix  = color.Bold.Sprintf("[") + color.LightBlue.Sprintf(prefixNotice) + color.Bold.Sprintf("]")
)

var colorMap = map[string]color.Color{
	"RED":           color.Red,
	"GREEN":         color.Green,
	"YELLOW":        color.Yellow,
	"BLUE":          color.Blue,
	"MAGENTA":       color.Magenta,
	"CYAN":          color.Cyan,
	"WHITE":         color.White,
	"LIGHT_RED":     color.LightRed,
	"LIGHT_GREEN":   color.LightGreen,
	"LIGHT_YELLOW":  color.LightYellow,
	"LIGHT_BLUE":    color.LightBlue,
	"LIGHT_MAGENTA": color.LightMagenta,
	"LIGHT_CYAN":    color.LightCyan,
	"LIGHT_WHITE":   color.LightWhite,
	"GRAY":          color.Gray,
}

func GetNoticeMessagePrefix() string {
	return noticeMsgPrefix
}

func GetCustomMessagePrefix(prefix string, prefixColor string) string {
	c, exists := colorMap[strings.ToUpper(prefixColor)]
	if !exists {
		c = color.Green
	}
	return color.Bold.Sprintf("[") + c.Sprint(prefix) + color.Bold.Sprintf("]")
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
