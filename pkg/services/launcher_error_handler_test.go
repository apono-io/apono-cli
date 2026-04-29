package services

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestTerminalErrorHandler_writesMessage(t *testing.T) {
	var buf bytes.Buffer
	h := &terminalErrorHandler{out: &buf}

	if err := h.Handle(defaultErrorTitle, "session not found", ""); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, defaultErrorTitle) || !strings.Contains(got, "session not found") {
		t.Errorf("expected output to contain title and message, got %q", got)
	}
}

func TestTerminalErrorHandler_includesStderrTail(t *testing.T) {
	var buf bytes.Buffer
	h := &terminalErrorHandler{out: &buf}

	if err := h.Handle(defaultErrorTitle, "command failed", "boom on stderr"); err != nil {
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

	if !strings.Contains(buf.String(), defaultErrorTitle) {
		t.Errorf("expected default title 'Apono' when empty, got %q", buf.String())
	}
}

func fakeHandler() (h *headlessErrorHandler, capturedScript *string) {
	capturedScript = new(string)
	h = &headlessErrorHandler{
		displayDialog: func(script string) error {
			*capturedScript = script
			return nil
		},
	}
	return
}

func TestHeadlessErrorHandler_buildsDisplayDialogScript(t *testing.T) {
	h, capturedScript := fakeHandler()

	if err := h.Handle(defaultErrorTitle, "session not found", ""); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	script := *capturedScript
	if !strings.Contains(script, "display dialog") {
		t.Errorf("expected script to contain 'display dialog', got %q", script)
	}
	if !strings.Contains(script, `"session not found"`) {
		t.Errorf("expected script to contain quoted message, got %q", script)
	}
	if !strings.Contains(script, fmt.Sprintf("with title %q", defaultErrorTitle)) {
		t.Errorf("expected script to contain quoted title, got %q", script)
	}
	if !strings.Contains(script, `buttons {"OK"}`) {
		t.Errorf("expected single OK button, got %q", script)
	}
}

func TestHeadlessErrorHandler_omitsStderrTailFromDialog(t *testing.T) {
	h, capturedScript := fakeHandler()

	if err := h.Handle(defaultErrorTitle, "command failed", "boom on stderr"); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	script := *capturedScript
	if !strings.Contains(script, "command failed") {
		t.Errorf("expected dialog to contain authored message, got %q", script)
	}
	if strings.Contains(script, "boom on stderr") {
		t.Errorf("dialog must not surface raw stderr to end users, got %q", script)
	}
}

func TestHeadlessErrorHandler_emptyTitleDefaultsToApono(t *testing.T) {
	h, capturedScript := fakeHandler()

	if err := h.Handle("", "msg", ""); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	if !strings.Contains(*capturedScript, fmt.Sprintf("with title %q", defaultErrorTitle)) {
		t.Errorf("expected default title 'Apono' in dialog, got %q", *capturedScript)
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
		got := appleScriptString(tc.in)
		if got != tc.want {
			t.Errorf("appleScriptString(%q): got %q, want %q", tc.in, got, tc.want)
		}
	}
}

