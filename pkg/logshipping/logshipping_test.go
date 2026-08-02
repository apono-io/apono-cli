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

func TestRedactForeignPaths(t *testing.T) {
	cases := []struct {
		name string
		text string
		want string
	}{
		{
			name: "a path inside an app bundle keeps only its last segment",
			text: "cd: /Applications/pgAdmin 4.app/Contents/Resources/web: No such file or directory",
			want: "cd: …/web: No such file or directory",
		},
		{
			name: "a working directory is reduced to the file name",
			text: `psql: could not open file "/Users/semyon/work/acme-migration/schema.sql"`,
			want: `psql: could not open file "…/schema.sql"`,
		},
		{
			name: "our own cache path survives whole",
			text: "read cache file: open /Users/semyon/.apono/cache/sess-1: no such file",
			want: "read cache file: open /Users/semyon/.apono/cache/sess-1: no such file",
		},
		{
			name: "our own launch script survives whole",
			text: "write launch script: open /var/folders/xy/abc/T/apono-launch-99.sh: denied",
			want: "write launch script: open /var/folders/xy/abc/T/apono-launch-99.sh: denied",
		},
		{
			name: "text without a path is untouched",
			text: `role "u" does not exist, try either/or`,
			want: `role "u" does not exist, try either/or`,
		},
		{
			name: "a missing binary reads the same",
			text: "sh: psql: command not found",
			want: "sh: psql: command not found",
		},
		{
			name: "prose after a path is not swallowed",
			text: "open /Users/x/a.txt failed because reasons",
			want: "open …/a.txt failed because reasons",
		},
		{
			name: "every path in the line is handled",
			text: "cp /Users/x/dev/one.sql /Users/x/dev/two.sql",
			want: "cp …/one.sql …/two.sql",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := redactForeignPaths(tc.text); got != tc.want {
				t.Errorf("redactForeignPaths() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRedactPaths_foreignPathsGoFirstThenHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got := redactPaths("open " + home + "/.apono/cache/sess-1 and " + home + "/work/acme/notes.md")

	want := "open ~/.apono/cache/sess-1 and …/notes.md"
	if got != want {
		t.Errorf("redactPaths() = %q, want %q", got, want)
	}
}

func TestSanitizeValues_doesNotMutateInput(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	in := map[string]string{"error": "open " + home + "/x"}
	sanitizeValues(in)

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
