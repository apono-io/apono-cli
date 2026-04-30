package terminal

import (
	"os"
	"strings"
	"testing"
)

func TestWriteLaunchScript_writesShebangAndKeepAlive(t *testing.T) {
	path, err := writeLaunchScript(`echo hi`)
	if err != nil {
		t.Fatalf("writeLaunchScript: %v", err)
	}
	defer os.Remove(path)

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read script: %v", err)
	}
	got := string(body)
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

func TestWriteLaunchScript_isExecutable(t *testing.T) {
	path, err := writeLaunchScript(`true`)
	if err != nil {
		t.Fatalf("writeLaunchScript: %v", err)
	}
	defer os.Remove(path)

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("expected executable bits set, got mode %v", info.Mode())
	}
}

func TestWriteLaunchScript_passesArbitraryCommandVerbatim(t *testing.T) {
	command := `psql "host=foo user=bar password='quux'" -c "select * from t;"`
	path, err := writeLaunchScript(command)
	if err != nil {
		t.Fatalf("writeLaunchScript: %v", err)
	}
	defer os.Remove(path)

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read script: %v", err)
	}
	if !strings.Contains(string(body), command) {
		t.Errorf("expected command verbatim in script, got:\n%s", string(body))
	}
}

func TestTerminalAppScript_format(t *testing.T) {
	got := terminalAppScript("/tmp/foo.sh")
	for _, want := range []string{`osascript`, `Terminal`, `do script`, `/tmp/foo.sh`, `activate`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in script, got %q", want, got)
		}
	}
}

func TestITermScript_format(t *testing.T) {
	got := iTermScript("/tmp/foo.sh")
	for _, want := range []string{`osascript`, `iTerm`, `create window with default profile command`, `/tmp/foo.sh`, `activate`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in script, got %q", want, got)
		}
	}
}
