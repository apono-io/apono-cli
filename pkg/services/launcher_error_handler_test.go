package services

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTerminalErrorHandler_writesMessage(t *testing.T) {
	var buf bytes.Buffer
	h := &terminalErrorHandler{out: &buf}

	if err := h.Handle("Apono", "session not found", ""); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "Apono") || !strings.Contains(got, "session not found") {
		t.Errorf("expected output to contain title and message, got %q", got)
	}
}

func TestTerminalErrorHandler_includesStderrTail(t *testing.T) {
	var buf bytes.Buffer
	h := &terminalErrorHandler{out: &buf}

	if err := h.Handle("Apono", "command failed", "boom on stderr"); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "boom on stderr") {
		t.Errorf("expected stderr tail in output, got %q", got)
	}
}

func TestTerminalErrorHandler_emptyTitleDefaultsToApono(t *testing.T) {
	var buf bytes.Buffer
	h := &terminalErrorHandler{out: &buf}

	if err := h.Handle("", "msg", ""); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	if !strings.Contains(buf.String(), "Apono") {
		t.Errorf("expected default title 'Apono' when empty, got %q", buf.String())
	}
}

func TestHeadlessErrorHandler_writesLog(t *testing.T) {
	logDir := t.TempDir()
	frozenTime := time.Date(2026, 4, 29, 10, 30, 0, 0, time.UTC)
	var capturedArgs []string

	h := &headlessErrorHandler{
		osascript: func(args ...string) error {
			capturedArgs = args
			return nil
		},
		logDir: logDir,
		now:    func() time.Time { return frozenTime },
	}

	if err := h.Handle("Apono", "session not found", ""); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	logBytes, err := os.ReadFile(filepath.Join(logDir, "launcher.log")) //nolint:gosec // logDir is t.TempDir()
	if err != nil {
		t.Fatalf("expected log file to exist: %v", err)
	}
	logLine := string(logBytes)

	if !strings.Contains(logLine, "2026-04-29T10:30:00Z") {
		t.Errorf("expected RFC3339 timestamp in log, got %q", logLine)
	}
	if !strings.Contains(logLine, "Apono") {
		t.Errorf("expected title in log, got %q", logLine)
	}
	if !strings.Contains(logLine, "session not found") {
		t.Errorf("expected message in log, got %q", logLine)
	}
	if len(capturedArgs) == 0 {
		t.Error("expected osascript to be invoked")
	}
}

func TestHeadlessErrorHandler_callsOsascriptWithDialog(t *testing.T) {
	var capturedArgs []string
	h := &headlessErrorHandler{
		osascript: func(args ...string) error {
			capturedArgs = args
			return nil
		},
		logDir: t.TempDir(),
		now:    time.Now,
	}

	if err := h.Handle("Apono", "session not found", ""); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	if len(capturedArgs) != 2 || capturedArgs[0] != "-e" {
		t.Fatalf("expected osascript called with [-e <script>], got %v", capturedArgs)
	}
	script := capturedArgs[1]
	if !strings.Contains(script, "display dialog") {
		t.Errorf("expected script to contain 'display dialog', got %q", script)
	}
	if !strings.Contains(script, `"session not found"`) {
		t.Errorf("expected script to contain quoted message, got %q", script)
	}
	if !strings.Contains(script, `with title "Apono"`) {
		t.Errorf("expected script to contain quoted title, got %q", script)
	}
	if !strings.Contains(script, `buttons {"OK"}`) {
		t.Errorf("expected single OK button, got %q", script)
	}
}

func TestHeadlessErrorHandler_includesStderrTailInDialog(t *testing.T) {
	var capturedArgs []string
	h := &headlessErrorHandler{
		osascript: func(args ...string) error {
			capturedArgs = args
			return nil
		},
		logDir: t.TempDir(),
		now:    time.Now,
	}

	if err := h.Handle("Apono", "command failed", "boom on stderr"); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	script := capturedArgs[1]
	if !strings.Contains(script, "command failed") {
		t.Errorf("expected dialog to contain message, got %q", script)
	}
	if !strings.Contains(script, "boom on stderr") {
		t.Errorf("expected dialog to contain stderr tail, got %q", script)
	}
}

func TestHeadlessErrorHandler_truncatesLongStderrTail(t *testing.T) {
	var capturedArgs []string
	h := &headlessErrorHandler{
		osascript: func(args ...string) error {
			capturedArgs = args
			return nil
		},
		logDir: t.TempDir(),
		now:    time.Now,
	}

	longStderr := strings.Repeat("x", 1000)
	if err := h.Handle("Apono", "msg", longStderr); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	script := capturedArgs[1]
	if strings.Contains(script, strings.Repeat("x", 1000)) {
		t.Errorf("expected stderr tail to be truncated in dialog, got %q", script)
	}
	if !strings.Contains(script, "…") {
		t.Errorf("expected truncation marker in dialog, got %q", script)
	}
}

func TestHeadlessErrorHandler_logEscapesNewlinesInStderr(t *testing.T) {
	logDir := t.TempDir()
	h := &headlessErrorHandler{
		osascript: func(args ...string) error { return nil },
		logDir:    logDir,
		now:       time.Now,
	}

	if err := h.Handle("Apono", "msg", "line1\nline2\nline3"); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	logBytes, err := os.ReadFile(filepath.Join(logDir, "launcher.log")) //nolint:gosec // logDir is t.TempDir()
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	logStr := string(logBytes)

	if strings.Count(logStr, "\n") != 1 {
		t.Errorf("expected single trailing newline in log line, got %q", logStr)
	}
	if !strings.Contains(logStr, "line1⏎line2⏎line3") {
		t.Errorf("expected stderr newlines replaced with ⏎, got %q", logStr)
	}
}

func TestHeadlessErrorHandler_emptyTitleDefaultsToApono(t *testing.T) {
	logDir := t.TempDir()
	var capturedArgs []string
	h := &headlessErrorHandler{
		osascript: func(args ...string) error {
			capturedArgs = args
			return nil
		},
		logDir: logDir,
		now:    time.Now,
	}

	if err := h.Handle("", "msg", ""); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	logBytes, _ := os.ReadFile(filepath.Join(logDir, "launcher.log")) //nolint:gosec // logDir is t.TempDir()
	if !strings.Contains(string(logBytes), "Apono") {
		t.Errorf("expected default title 'Apono' in log, got %q", string(logBytes))
	}

	script := capturedArgs[1]
	if !strings.Contains(script, `with title "Apono"`) {
		t.Errorf("expected default title 'Apono' in dialog, got %q", script)
	}
}

func TestHeadlessErrorHandler_swallowsLogDirError(t *testing.T) {
	// Empty logDir disables logging entirely; Handle should still succeed.
	var osascriptCalled bool
	h := &headlessErrorHandler{
		osascript: func(args ...string) error {
			osascriptCalled = true
			return nil
		},
		logDir: "",
		now:    time.Now,
	}

	if err := h.Handle("Apono", "msg", ""); err != nil {
		t.Fatalf("Handle should not error when log dir disabled: %v", err)
	}
	if !osascriptCalled {
		t.Error("expected osascript to still be called when log dir disabled")
	}
}

func TestApplescriptString_escapesQuotesAndBackslashes(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`hello`, `"hello"`},
		{`he said "hi"`, `"he said \"hi\""`},
		{`path\to\file`, `"path\\to\\file"`},
		{`mix \ and "`, `"mix \\ and \""`},
		{``, `""`},
	}
	for _, tc := range cases {
		got := applescriptString(tc.in)
		if got != tc.want {
			t.Errorf("applescriptString(%q): got %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestTruncateStderrTail(t *testing.T) {
	cases := []struct {
		name string
		in   string
		n    int
		want string
	}{
		{"shorter than limit", "abc", 10, "abc"},
		{"equal to limit", "abcde", 5, "abcde"},
		{"longer than limit", "abcdefghij", 5, "…fghij"},
		{"zero limit returns input unchanged", "abc", 0, "abc"},
		{"negative limit returns input unchanged", "abc", -1, "abc"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := truncateStderrTail(tc.in, tc.n)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
