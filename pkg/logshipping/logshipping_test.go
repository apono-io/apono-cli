package logshipping

import "testing"

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
