package urihandler

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

func TestDrainLog_everyKnownLevelPassesThrough(t *testing.T) {
	levels := []string{
		logshipping.LevelTrace,
		logshipping.LevelDebug,
		logshipping.LevelInfo,
		logshipping.LevelWarn,
		logshipping.LevelError,
	}

	var body strings.Builder
	for _, level := range levels {
		fmt.Fprintf(&body, "%s\tmessage\n", level)
	}
	path := writeLog(t, body.String())

	lines, err := DrainLog(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != len(levels) {
		t.Fatalf("expected %d lines, got %d: %+v", len(levels), len(lines), lines)
	}
	for i, level := range levels {
		if lines[i] != (LogLine{Level: level, Message: "message"}) {
			t.Errorf("line %d = %+v, want level %q with the message split off", i, lines[i], level)
		}
	}
}

func TestDrainLog_unknownLevel_keepsWholeLineAsMessage(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{"invented severity", "CRITICAL\tdisk on fire"},
		{"lowercase severity", "info\tnot our spelling"},
		{"prose before a tab", "some prefix\ttrailing text"},
		{"blank before a tab", "\ttrailing text"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines, err := DrainLog(writeLog(t, tc.raw+"\n"))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(lines) != 1 {
				t.Fatalf("expected 1 line, got %d: %+v", len(lines), lines)
			}
			if lines[0] != (LogLine{Level: logshipping.LevelInfo, Message: tc.raw}) {
				t.Errorf("got %+v, want INFO carrying the whole line %q", lines[0], tc.raw)
			}
		})
	}
}

func TestDrainLog_overLineCap_keepsNewestAndCountsTheRest(t *testing.T) {
	const total = maxDrainLines + 17

	var body strings.Builder
	for i := range total {
		fmt.Fprintf(&body, "INFO\tline-%d\n", i)
	}
	path := writeLog(t, body.String())

	lines, err := DrainLog(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != maxDrainLines+1 {
		t.Fatalf("expected %d lines (cap plus one notice), got %d", maxDrainLines+1, len(lines))
	}
	if lines[0].Level != logshipping.LevelInfo || !strings.Contains(lines[0].Message, "17") {
		t.Errorf("expected an INFO notice counting the 17 dropped lines, got %+v", lines[0])
	}
	if lines[1].Message != "line-17" {
		t.Errorf("expected the oldest surviving line to be line-17, got %q", lines[1].Message)
	}
	if lines[len(lines)-1].Message != fmt.Sprintf("line-%d", total-1) {
		t.Errorf("expected the newest line to survive, got %q", lines[len(lines)-1].Message)
	}
}

func TestDrainLog_atLineCap_addsNoNotice(t *testing.T) {
	var body strings.Builder
	for i := range maxDrainLines {
		fmt.Fprintf(&body, "INFO\tline-%d\n", i)
	}
	path := writeLog(t, body.String())

	lines, err := DrainLog(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != maxDrainLines {
		t.Fatalf("expected exactly %d lines with no notice, got %d", maxDrainLines, len(lines))
	}
	if lines[0].Message != "line-0" {
		t.Errorf("expected the first line to survive untouched, got %q", lines[0].Message)
	}
}

func TestDrainLog_overByteCap_countsDroppedLines(t *testing.T) {
	const entry = "INFO\tx\n"

	body := strings.Repeat(entry, (maxDrainBytes/len(entry))+100)
	path := writeLog(t, body)

	lines, err := DrainLog(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) == 0 {
		t.Fatal("expected a truncation notice, got no lines")
	}

	wantDropped := (len(body) - maxDrainBytes) / len(entry)
	if lines[0].Level != logshipping.LevelInfo || !strings.Contains(lines[0].Message, strconv.Itoa(wantDropped)) {
		t.Fatalf("expected an INFO notice counting %d dropped lines, got %+v", wantDropped, lines[0])
	}
}
