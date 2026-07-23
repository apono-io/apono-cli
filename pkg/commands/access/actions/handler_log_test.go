package actions

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// Without an authenticated client in context, shipHandlerLog must NOT drain the
// file — draining empties it, and shipping would be a no-op, so the lines would
// be lost. This guards the "never empty what we can't ship" invariant.
func TestShipHandlerLog_noClient_doesNotDrainFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	logDir := filepath.Join(home, "Library", "Application Support", "apono-cli")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	logPath := filepath.Join(logDir, "handler.log")
	if err := os.WriteFile(logPath, []byte("INFO\thello\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	shipHandlerLog(context.Background()) // no client in context

	data, err := os.ReadFile(filepath.Clean(logPath))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("file was drained without a client — handler logs were lost")
	}
}
