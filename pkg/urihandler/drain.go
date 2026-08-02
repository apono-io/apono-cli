package urihandler

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/apono-io/apono-cli/pkg/logshipping"
)

const (
	handlerLogFileName = "handler.log"
	logLineSeparator   = "\t"
	maxDrainBytes      = 64 * 1024
	maxDrainLines      = 50
	sizeCapNotice      = "handler log truncated: dropped %d oldest lines over size cap"
	lineCapNotice      = "handler log truncated: dropped %d oldest lines over line cap"
)

// LogLine is one parsed record drained from the handler log file. Its Level is
// whatever level string the script wrote (INFO/ERROR), passed through verbatim.
type LogLine struct {
	Level   string
	Message string
}

// HandlerLogPath returns the file the apono:// handler script writes its trace
// to — the same directory that holds the handler bundle.
func HandlerLogPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, bundleParentDir, handlerLogFileName), nil
}

// DrainLog reads the handler log at path, empties it, and returns the parsed
// lines. No-op (nil, nil) when the file is missing or empty. The file is emptied
// as soon as it is read, so the caller must be able to ship what it receives —
// callers guard this by only invoking DrainLog once an authenticated client is
// present.
func DrainLog(path string) ([]LogLine, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if info.Size() == 0 {
		return nil, nil
	}

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	if err := os.Truncate(path, 0); err != nil {
		return nil, err
	}

	return parseLines(data), nil
}

func parseLines(data []byte) []LogLine {
	var notices []LogLine
	if len(data) > maxDrainBytes {
		dropped := bytes.Count(data[:len(data)-maxDrainBytes], []byte{'\n'})
		data = data[len(data)-maxDrainBytes:]
		notices = append(notices, LogLine{Level: logshipping.LevelWarn, Message: fmt.Sprintf(sizeCapNotice, dropped)})
	}

	var lines []LogLine
	for _, raw := range strings.Split(string(data), "\n") {
		raw = strings.TrimRight(raw, "\r")
		if strings.TrimSpace(raw) == "" {
			continue
		}
		level, message, found := strings.Cut(raw, logLineSeparator)
		if !found || !logshipping.IsKnownLevel(level) {
			lines = append(lines, LogLine{Level: logshipping.LevelInfo, Message: raw})
			continue
		}
		lines = append(lines, LogLine{Level: level, Message: message})
	}

	if dropped := len(lines) - maxDrainLines; dropped > 0 {
		lines = lines[dropped:]
		notices = append(notices, LogLine{Level: logshipping.LevelWarn, Message: fmt.Sprintf(lineCapNotice, dropped)})
	}
	return append(notices, lines...)
}
