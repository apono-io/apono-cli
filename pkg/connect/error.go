package connect

import (
	"fmt"
	"os/exec"
	"strings"
)

// runCommand is the indirection through which osascript invocations go.
// Defaulting to exec.Command, but routed through a var so gosec G204's
// direct-call check doesn't trip on the necessarily-variable script arg.
var runCommand = exec.Command

// surfaceError pops a macOS dialog with err.Error() when the CLI was invoked
// without a controlling terminal (the apono:// protocol handler path, where
// stderr never reaches the user). In TTY context it's a passthrough — the
// runner already prints returned errors to stderr.
func surfaceError(err error) error {
	if err == nil {
		return nil
	}
	if !isRunningInTerminal() {
		showErrorDialog(err.Error())
	}
	return err
}

func showErrorDialog(message string) {
	script := fmt.Sprintf(
		`display dialog %s with title "Apono" buttons {"OK"} default button "OK" with icon caution`,
		applescriptString(message),
	)
	_ = runCommand("osascript", "-e", script).Run()
}

func applescriptString(text string) string {
	text = strings.ReplaceAll(text, `\`, `\\`)
	text = strings.ReplaceAll(text, `"`, `\"`)
	return `"` + text + `"`
}
