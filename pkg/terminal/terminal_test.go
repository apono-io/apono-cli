package terminal

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestWriteLaunchScriptTo_includesShebangSelfDeleteAndKeepAlive(t *testing.T) {
	var buf bytes.Buffer
	if err := writeLaunchScriptTo(&buf, `echo hi`); err != nil {
		t.Fatalf("writeLaunchScriptTo: %v", err)
	}
	got := buf.String()
	for _, want := range []string{
		"#!/bin/zsh",
		`rm -- "$0"`,
		"echo hi",
		"exec /bin/zsh -l",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("script body missing %q, got:\n%s", want, got)
		}
	}
}

func TestWriteLaunchScriptTo_passesArbitraryCommandVerbatim(t *testing.T) {
	command := `psql "host=foo user=bar password='quux'" -c "select * from t;"`
	var buf bytes.Buffer
	if err := writeLaunchScriptTo(&buf, command); err != nil {
		t.Fatalf("writeLaunchScriptTo: %v", err)
	}
	if !strings.Contains(buf.String(), command) {
		t.Errorf("expected command verbatim in script, got:\n%s", buf.String())
	}
}

func TestWriteLaunchScript_returnsExistingPath(t *testing.T) {
	path, err := writeLaunchScript(`true`)
	if err != nil {
		t.Fatalf("writeLaunchScript: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected script file at %q, got stat error: %v", path, err)
	}
}

func TestTerminalAppScript_format(t *testing.T) {
	got := terminalAppScript("/tmp/foo.sh")
	for _, want := range []string{`osascript`, `Terminal`, `do script`, `/bin/zsh /tmp/foo.sh`, `activate`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in script, got %q", want, got)
		}
	}
}

func TestITermScript_format(t *testing.T) {
	got := iTermScript("/tmp/foo.sh")
	for _, want := range []string{`osascript`, `iTerm`, `create window with default profile command`, `/bin/zsh /tmp/foo.sh`, `activate`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in script, got %q", want, got)
		}
	}
}
