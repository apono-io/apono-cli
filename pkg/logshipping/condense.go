package logshipping

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	maxFieldBytes    = 4 * 1024
	repeatSuffix     = " [repeated %d times]"
	truncationSuffix = "…[truncated]"
)

func condense(text string) string {
	return truncate(collapseRepeats(text), maxFieldBytes)
}

func collapseRepeats(text string) string {
	var kept []string
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) != "" {
			kept = append(kept, line)
		}
	}

	var collapsed []string
	for i := 0; i < len(kept); {
		run := i + 1
		for run < len(kept) && kept[run] == kept[i] {
			run++
		}
		if repeats := run - i; repeats > 1 {
			collapsed = append(collapsed, kept[i]+fmt.Sprintf(repeatSuffix, repeats))
		} else {
			collapsed = append(collapsed, kept[i])
		}
		i = run
	}
	return strings.Join(collapsed, "\n")
}

func truncate(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	cut := limit
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	return text[:cut] + truncationSuffix
}
