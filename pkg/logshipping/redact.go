package logshipping

import (
	"os"
	"strings"
)

const homeDirMarker = "~"

func redactHomeDir(text string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" || home == string(os.PathSeparator) {
		return text
	}
	return strings.ReplaceAll(text, home, homeDirMarker)
}

func redactValues(fields map[string]string) map[string]string {
	if fields == nil {
		return nil
	}
	redacted := make(map[string]string, len(fields))
	for key, value := range fields {
		redacted[key] = redactHomeDir(value)
	}
	return redacted
}
