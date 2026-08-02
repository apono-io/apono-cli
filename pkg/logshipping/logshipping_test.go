package logshipping

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRedactHomeDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cases := []struct {
		name string
		text string
		want string
	}{
		{
			name: "path under home collapses to marker",
			text: "open " + filepath.Join(home, ".apono", "cache", "sess-1") + ": no such file or directory",
			want: "open " + filepath.Join(homeDirMarker, ".apono", "cache", "sess-1") + ": no such file or directory",
		},
		{
			name: "every occurrence is replaced",
			text: home + " and " + home,
			want: homeDirMarker + " and " + homeDirMarker,
		},
		{
			name: "text without the home path is untouched",
			text: "connection refused",
			want: "connection refused",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := redactHomeDir(tc.text); got != tc.want {
				t.Errorf("redactHomeDir(%q) = %q, want %q", tc.text, got, tc.want)
			}
		})
	}
}

func TestRedactHomeDir_rootHomeLeavesTextIntact(t *testing.T) {
	t.Setenv("HOME", string(filepath.Separator))

	text := "/etc/hosts and /var/log"
	if got := redactHomeDir(text); got != text {
		t.Errorf("redactHomeDir(%q) = %q, want it unchanged when home is the root", text, got)
	}
}

func TestBuildLogEntry_redactsHomeDirFromMessageAndFields(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	secretPath := filepath.Join(home, ".apono", "cache", "sess-1")

	entry := buildLogEntry(CallerCLI, LevelError, "read cache file: open "+secretPath, map[string]string{
		"error": "resolve credentials: open " + secretPath,
	})

	if strings.Contains(entry.Message, home) {
		t.Errorf("message still carries the home path: %q", entry.Message)
	}
	if strings.Contains(entry.Fields["error"], home) {
		t.Errorf("field still carries the home path: %q", entry.Fields["error"])
	}
	if !strings.Contains(entry.Fields["error"], ".apono") {
		t.Errorf("expected the rest of the path to survive redaction, got %q", entry.Fields["error"])
	}
}

func TestRedactValues_doesNotMutateInput(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	in := map[string]string{"error": "open " + home + "/x"}
	redactValues(in)

	if !strings.Contains(in["error"], home) {
		t.Errorf("input map was mutated: %q", in["error"])
	}
}

func TestBuildLogEntry_threadsCallerLevelMessageAndFields(t *testing.T) {
	entry := buildLogEntry(CallerHandler, LevelError, "boom", map[string]string{"k": "v"})

	if got := entry.GetCaller(); got != CallerHandler {
		t.Errorf("caller = %q, want %q", got, CallerHandler)
	}
	if entry.Level != LevelError {
		t.Errorf("level = %q, want %q", entry.Level, LevelError)
	}
	if entry.Message != "boom" {
		t.Errorf("message = %q, want %q", entry.Message, "boom")
	}
	if entry.Fields["k"] != "v" {
		t.Errorf("fields[k] = %q, want %q", entry.Fields["k"], "v")
	}
	if _, ok := entry.Fields[fieldCLIVersion]; !ok {
		t.Errorf("expected %q field to be present", fieldCLIVersion)
	}
	if entry.SessionId != sessionID {
		t.Errorf("session id = %q, want %q", entry.SessionId, sessionID)
	}
}

func TestWithCLIVersion_doesNotMutateInput(t *testing.T) {
	in := map[string]string{"a": "1"}
	out := withCLIVersion(in)

	if _, ok := in[fieldCLIVersion]; ok {
		t.Errorf("input map was mutated with %q", fieldCLIVersion)
	}
	if out["a"] != "1" {
		t.Errorf("out[a] = %q, want %q", out["a"], "1")
	}
}
