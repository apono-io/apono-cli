package connect

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// surfaceError pops a macOS dialog with err.Error() when the CLI was invoked
// without a controlling terminal (the apono:// protocol handler path, where
// stderr never reaches the user). In TTY context it's a passthrough — the
// runner already prints returned errors to stderr.
func surfaceError(err error) error {
	if err == nil {
		return nil
	}
	if !isRunningInTerminal() {
		runOsaScript(buildErrorDialogScript(err.Error()))
	}
	return err
}

func buildErrorDialogScript(message string) string {
	return fmt.Sprintf(
		`display dialog %s with title "Apono" buttons {"OK"} default button "OK" with icon caution`,
		applescriptString(message),
	)
}

func runOsaScript(script string) {
	_ = exec.CommandContext(context.Background(), "osascript", "-e", script).Run()
}

func applescriptString(text string) string {
	text = strings.ReplaceAll(text, `\`, `\\`)
	text = strings.ReplaceAll(text, `"`, `\"`)
	return `"` + text + `"`
}
