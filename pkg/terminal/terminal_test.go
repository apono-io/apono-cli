package terminal

import (
	"strings"
	"testing"
)

func TestBuildLaunchCommand_escapesQuotesAndBackslashes(t *testing.T) {
	got := BuildLaunchCommand(`echo "hi" \n`)
	if !strings.Contains(got, `\"hi\"`) {
		t.Errorf("expected double quotes to be escaped, got %q", got)
	}
	if !strings.Contains(got, `\\n`) {
		t.Errorf("expected backslashes to be escaped, got %q", got)
	}
	if !strings.Contains(got, `osascript`) || !strings.Contains(got, `Terminal`) {
		t.Errorf("expected osascript Terminal call, got %q", got)
	}
}

func TestBuildLaunchCommand_escapesSingleQuotes(t *testing.T) {
	got := BuildLaunchCommand(`echo 'hi'`)

	if !strings.Contains(got, `'\''hi'\''`) {
		t.Errorf("expected single quotes to be shell-escaped, got %q", got)
	}
}
