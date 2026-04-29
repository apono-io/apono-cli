package services

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	headlessLogFileName     = "launcher.log"
	headlessStderrTailLimit = 500
)

type ErrorHandler interface {
	Handle(title, message, stderrTail string) error
}

func ChooseErrorHandler(cobraCmd *cobra.Command) ErrorHandler {
	return chooseErrorHandler(cobraCmd)
}

func chooseErrorHandler(cobraCmd *cobra.Command) ErrorHandler {
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
		title = "Apono"
	}
	if stderrTail != "" {
		_, err := fmt.Fprintf(t.out, "%s: %s\n%s\n", title, message, stderrTail)
		return err
	}
	_, err := fmt.Fprintf(t.out, "%s: %s\n", title, message)
	return err
}

// headlessErrorHandler surfaces errors to the user via osascript dialog when
// the CLI is invoked without a TTY (e.g. by the Apono Connect.app protocol
// handler). All errors are also appended to ~/Library/Logs/apono/launcher.log
// regardless of whether the dialog succeeds.
type headlessErrorHandler struct {
	displayDialog func(script string) error
	appendLog     func(line string)
	now           func() time.Time
}

func newHeadlessErrorHandler() *headlessErrorHandler {
	return &headlessErrorHandler{
		displayDialog: runOsascript,
		appendLog:     appendToDefaultLogFile,
		now:           time.Now,
	}
}

func (h *headlessErrorHandler) Handle(title, message, stderrTail string) error {
	if title == "" {
		title = "Apono"
	}

	h.writeLog(title, message, stderrTail)

	body := message
	if stderrTail != "" {
		body = fmt.Sprintf("%s\n\n%s", message, truncateStderrTail(stderrTail, headlessStderrTailLimit))
	}

	script := fmt.Sprintf(
		`display dialog %s with title %s buttons {"OK"} default button "OK" with icon caution`,
		applescriptString(body),
		applescriptString(title),
	)
	return h.displayDialog(script)
}

func (h *headlessErrorHandler) writeLog(title, message, stderrTail string) {
	if h.appendLog == nil {
		return
	}
	line := fmt.Sprintf("%s\t%s\t%s", h.now().UTC().Format(time.RFC3339), title, message)
	if stderrTail != "" {
		line = fmt.Sprintf("%s\tstderr=%s", line, strings.ReplaceAll(stderrTail, "\n", "⏎"))
	}
	h.appendLog(line)
}

// appendToDefaultLogFile is the production sink: open ~/Library/Logs/apono/launcher.log,
// append the line, close. Errors are swallowed — logging is best-effort and
// must never block the user-facing dialog.
func appendToDefaultLogFile(line string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, "Library", "Logs", "apono")
	if err = os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	logPath := filepath.Clean(filepath.Join(dir, headlessLogFileName))
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = fmt.Fprintln(f, line)
}

func runOsascript(script string) error {
	return exec.Command("osascript", "-e", script).Run()
}

// applescriptString quotes a Go string as an AppleScript double-quoted literal.
// Escapes \ and " — sufficient for `display dialog` text and titles.
func applescriptString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

// truncateStderrTail keeps the last n characters of s, prefixing with "…"
// if truncation occurred.
func truncateStderrTail(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n:]
}
