package terminal

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func IsRunning(in io.Reader) bool {
	f, ok := in.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func BuildLaunchCommand(command string) (string, error) {
	scriptPath, err := writeLaunchScript(command)
	if err != nil {
		return "", fmt.Errorf("write launch script: %w", err)
	}
	if hasITerm() {
		return iTermScript(scriptPath), nil
	}
	return terminalAppScript(scriptPath), nil
}

const launchScriptTemplate = `#!/bin/zsh
rm -- "$0"
%s
exec /bin/zsh -l
`

func writeLaunchScript(command string) (string, error) {
	f, err := os.CreateTemp("", "apono-launch-*.sh")
	if err != nil {
		return "", err
	}
	if _, err := fmt.Fprintf(f, launchScriptTemplate, command); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	if err := os.Chmod(f.Name(), 0o755); err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func hasITerm() bool {
	paths := []string{"/Applications/iTerm.app"}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, "Applications", "iTerm.app"))
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

func terminalAppScript(scriptPath string) string {
	return fmt.Sprintf(
		`osascript -e 'tell application "Terminal" to do script "%s"' -e 'tell application "Terminal" to activate'`,
		scriptPath,
	)
}

func iTermScript(scriptPath string) string {
	return fmt.Sprintf(
		`osascript -e 'tell application "iTerm" to create window with default profile command "%s"' -e 'tell application "iTerm" to activate'`,
		scriptPath,
	)
}
