package services

import (
	"bytes"
	"strings"
	"testing"
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
