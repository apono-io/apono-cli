package services

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

type ErrorHandler interface {
	Handle(title, message, stderrTail string) error
}

func chooseErrorHandler(cobraCmd *cobra.Command) ErrorHandler {
	if isTTY() {
		return &terminalErrorHandler{out: cobraCmd.ErrOrStderr()}
	}
	return &headlessErrorHandler{}
}

type terminalErrorHandler struct {
	out io.Writer
}

func (t *terminalErrorHandler) Handle(title, message, stderrTail string) error {
	if title == "" {
		title = "Apono"
	}
	if stderrTail != "" {
		_, err := fmt.Fprintf(t.out, "%s: %s\n%s\n", title, message, stderrTail)
		return err
	}
	_, err := fmt.Fprintf(t.out, "%s: %s\n", title, message)
	return err
}

// TODO(DVL-8794): osascript display-dialog + log file under ~/Library/Logs/apono/.
type headlessErrorHandler struct{}

func (h *headlessErrorHandler) Handle(_, _, _ string) error {
	return nil
}
