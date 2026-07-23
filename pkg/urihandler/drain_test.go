package urihandler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apono-io/apono-cli/pkg/logshipping"
)

func writeLog(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "handler.log")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	return path
}

func TestDrainLog_missingFile_isNoop(t *testing.T) {
	lines, err := DrainLog(filepath.Join(t.TempDir(), "absent.log"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lines != nil {
		t.Errorf("expected nil lines for missing file, got %v", lines)
	}
}

func TestDrainLog_emptyFile_isNoop(t *testing.T) {
	path := writeLog(t, "")
	lines, err := DrainLog(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lines != nil {
		t.Errorf("expected nil lines for empty file, got %v", lines)
	}
}

func TestDrainLog_parsesLinesAndEmptiesFile(t *testing.T) {
	path := writeLog(t, "INFO\treceived launch request\nERROR\tapono CLI not found code=127\n")

	lines, err := DrainLog(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != (LogLine{Level: logshipping.LevelInfo, Message: "received launch request"}) {
		t.Errorf("line 0 = %+v", lines[0])
	}
	if lines[1] != (LogLine{Level: logshipping.LevelError, Message: "apono CLI not found code=127"}) {
		t.Errorf("line 1 = %+v", lines[1])
	}

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected file emptied after drain, still has %d bytes", len(data))
	}
}

func TestDrainLog_malformedLineDefaultsToInfo(t *testing.T) {
	path := writeLog(t, "no tab in this line\n")
	lines, err := DrainLog(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 1 || lines[0].Level != logshipping.LevelInfo || lines[0].Message != "no tab in this line" {
		t.Errorf("expected INFO fallback with raw message, got %+v", lines)
	}
}

func TestDrainLog_overCapPrependsTruncationNotice(t *testing.T) {
	body := strings.Repeat("INFO\tx\n", (maxDrainBytes/7)+100) // 7 bytes per line, well over cap
	path := writeLog(t, body)

	lines, err := DrainLog(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) == 0 || lines[0].Level != logshipping.LevelWarn || !strings.Contains(lines[0].Message, "truncated") {
		t.Fatalf("expected first line to be a WARN truncation notice, got %+v", lines[0])
	}
}
