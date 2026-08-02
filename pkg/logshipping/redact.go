package logshipping

import (
	"os"
	"path"
	"regexp"
	"strings"
)

const (
	homeDirMarker    = "~"
	aponoPathMarker  = "apono"
	elidedPathPrefix = "…/"
)

var absolutePath = regexp.MustCompile(`/(?:[^/\s:"']+(?: [^/\s:"']+)*/)+[^/\s:"']*`)

func redactPaths(text string) string {
	return redactHomeDir(redactForeignPaths(text))
}

func redactForeignPaths(text string) string {
	return absolutePath.ReplaceAllStringFunc(text, func(match string) string {
		if strings.Contains(strings.ToLower(match), aponoPathMarker) {
			return match
		}
		return elidedPathPrefix + path.Base(match)
	})
}

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
		redacted[key] = redactPaths(value)
	}
	return redacted
}
