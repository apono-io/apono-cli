package services

import (
	"bytes"
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

// fakeHandler returns a headlessErrorHandler with both side channels (osascript
// dialog + log append) captured in test-owned variables. Callers inspect
// *capturedScript and *logBuf after Handle to verify behavior. The clock is
// frozen at 2026-04-29T10:30:00Z so log-line assertions can match a literal
// RFC3339 timestamp.
func fakeHandler(t *testing.T) (h *headlessErrorHandler, capturedScript *string, logBuf *bytes.Buffer) {
	t.Helper()
	capturedScript = new(string)
	logBuf = &bytes.Buffer{}
	h = &headlessErrorHandler{
		displayDialog: func(script string) error {
			*capturedScript = script
			return nil
		},
		appendLog: func(line string) {
			logBuf.WriteString(line)
			logBuf.WriteByte('\n')
		},
		now: func() time.Time { return time.Date(2026, 4, 29, 10, 30, 0, 0, time.UTC) },
	}
	return
}

func TestHeadlessErrorHandler_appendsLogLine(t *testing.T) {
	h, _, logBuf := fakeHandler(t)

	if err := h.Handle("Apono", "session not found", ""); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	logLine := logBuf.String()
	if !strings.Contains(logLine, "2026-04-29T10:30:00Z") {
		t.Errorf("expected RFC3339 timestamp in log, got %q", logLine)
	}
	if !strings.Contains(logLine, "Apono") {
		t.Errorf("expected title in log, got %q", logLine)
	}
	if !strings.Contains(logLine, "session not found") {
		t.Errorf("expected message in log, got %q", logLine)
	}
}

func TestHeadlessErrorHandler_buildsDisplayDialogScript(t *testing.T) {
	h, capturedScript, _ := fakeHandler(t)

	if err := h.Handle("Apono", "session not found", ""); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	script := *capturedScript
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
	h, capturedScript, _ := fakeHandler(t)

	if err := h.Handle("Apono", "command failed", "boom on stderr"); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	script := *capturedScript
	if !strings.Contains(script, "command failed") {
		t.Errorf("expected dialog to contain message, got %q", script)
	}
	if !strings.Contains(script, "boom on stderr") {
		t.Errorf("expected dialog to contain stderr tail, got %q", script)
	}
}

func TestHeadlessErrorHandler_truncatesLongStderrTail(t *testing.T) {
	h, capturedScript, _ := fakeHandler(t)

	longStderr := strings.Repeat("x", 1000)
	if err := h.Handle("Apono", "msg", longStderr); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	script := *capturedScript
	if strings.Contains(script, strings.Repeat("x", 1000)) {
		t.Errorf("expected stderr tail to be truncated in dialog, got %q", script)
	}
	if !strings.Contains(script, "…") {
		t.Errorf("expected truncation marker in dialog, got %q", script)
	}
}

func TestHeadlessErrorHandler_logEscapesNewlinesInStderr(t *testing.T) {
	h, _, logBuf := fakeHandler(t)

	if err := h.Handle("Apono", "msg", "line1\nline2\nline3"); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	logStr := logBuf.String()
	if strings.Count(logStr, "\n") != 1 {
		t.Errorf("expected single trailing newline in log line, got %q", logStr)
	}
	if !strings.Contains(logStr, "line1⏎line2⏎line3") {
		t.Errorf("expected stderr newlines replaced with ⏎, got %q", logStr)
	}
}

func TestHeadlessErrorHandler_emptyTitleDefaultsToApono(t *testing.T) {
	h, capturedScript, logBuf := fakeHandler(t)

	if err := h.Handle("", "msg", ""); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	if !strings.Contains(logBuf.String(), "Apono") {
		t.Errorf("expected default title 'Apono' in log, got %q", logBuf.String())
	}
	if !strings.Contains(*capturedScript, `with title "Apono"`) {
		t.Errorf("expected default title 'Apono' in dialog, got %q", *capturedScript)
	}
}

func TestHeadlessErrorHandler_skipsLoggingWhenSinkNil(t *testing.T) {
	var capturedScript string
	h := &headlessErrorHandler{
		displayDialog: func(script string) error {
			capturedScript = script
			return nil
		},
		appendLog: nil,
		now:       time.Now,
	}

	if err := h.Handle("Apono", "msg", ""); err != nil {
		t.Fatalf("Handle should not error when log sink nil: %v", err)
	}
	if capturedScript == "" {
		t.Error("expected dialog to still fire when log sink nil")
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
