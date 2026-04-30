package terminal

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func IsRunning(in io.Reader) bool {
	f, ok := in.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func BuildLaunchCommand(command string) string {
	escaped := strings.ReplaceAll(command, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	escaped = strings.ReplaceAll(escaped, `'`, `'\''`)

	return fmt.Sprintf(`osascript -e 'tell application "Terminal" to do script "%s"' -e 'tell application "Terminal" to activate'`, escaped)
}
