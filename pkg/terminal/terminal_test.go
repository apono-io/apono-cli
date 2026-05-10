package terminal

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestIsRunning_devNull_returnsFalse(t *testing.T) {
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open /dev/null: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })

	if IsRunning(f) {
		t.Errorf("IsRunning(/dev/null) = true, want false (regression: char-device check used to misclassify /dev/null as TTY)")
	}
}

func TestIsRunning_nonFile_returnsFalse(t *testing.T) {
	if IsRunning(&bytes.Buffer{}) {
		t.Error("IsRunning(*bytes.Buffer) = true, want false")
	}
}

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

func TestBuildTerminalAppLaunchCommand_format(t *testing.T) {
	got := buildTerminalAppLaunchCommand("/tmp/foo.sh")
	for _, want := range []string{`osascript`, `Terminal`, `do script`, `/bin/zsh /tmp/foo.sh`, `activate`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in script, got %q", want, got)
		}
	}
}

func TestBuildITermLaunchCommand_format(t *testing.T) {
	got := buildITermLaunchCommand("/tmp/foo.sh")
	for _, want := range []string{`osascript`, `iTerm`, `create window with default profile command`, `/bin/zsh /tmp/foo.sh`, `activate`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in script, got %q", want, got)
		}
	}
}
