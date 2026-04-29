package services

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

const defaultErrorTitle = "Apono"

type ErrorHandler interface {
	Handle(title, message, stderrTail string) error
}

func ChooseErrorHandler(cobraCmd *cobra.Command) ErrorHandler {
	if isTTY() {
		return &terminalErrorHandler{out: cobraCmd.ErrOrStderr()}
	}
	return newHeadlessErrorHandler()
}

type terminalErrorHandler struct {
	out io.Writer
}

func (t *terminalErrorHandler) Handle(title, message, stderrTail string) error {
	if title == "" {
		title = defaultErrorTitle
	}
	if stderrTail != "" {
		_, err := fmt.Fprintf(t.out, "%s: %s\n%s\n", title, message, stderrTail)
		return err
	}
	_, err := fmt.Fprintf(t.out, "%s: %s\n", title, message)
	return err
}

type headlessErrorHandler struct {
	displayDialog func(script string) error
}

func newHeadlessErrorHandler() *headlessErrorHandler {
	return &headlessErrorHandler{displayDialog: runOsascript}
}

func (h *headlessErrorHandler) Handle(title, message, _ string) error {
	if title == "" {
		title = defaultErrorTitle
	}
	script := fmt.Sprintf(
		`display dialog %s with title %s buttons {"OK"} default button "OK" with icon caution`,
		appleScriptString(message),
		appleScriptString(title),
	)
	return h.displayDialog(script)
}

func runOsascript(script string) error {
	return exec.Command("osascript", "-e", script).Run()
}

func appleScriptString(text string) string {
	text = strings.ReplaceAll(text, `\`, `\\`)
	text = strings.ReplaceAll(text, `"`, `\"`)
	return `"` + text + `"`
}

