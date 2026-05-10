package terminal

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"
)

const (
	terminalAppName = "Terminal"
	iTermAppName    = "iTerm"

	binZsh = "/bin/zsh"

	terminalAppLaunchTemplate = `osascript -e 'tell application "%s" to do script "%s %s"' -e 'tell application "%s" to activate'`
	iTermLaunchTemplate       = `osascript -e 'tell application "%s" to create window with default profile command "%s %s"' -e 'tell application "%s" to activate'`

	launchCommandPlaceholder = "__APONO_COMMAND__"
)

//go:embed scripts/launch.sh
var launchScriptTemplate string

func IsRunning(in io.Reader) bool {
	f, ok := in.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd())
}

func BuildLaunchCommand(command string) (string, error) {
	scriptPath, err := writeLaunchScript(command)
	if err != nil {
		return "", fmt.Errorf("write launch script: %w", err)
	}
	if hasITerm() {
		return buildITermLaunchCommand(scriptPath), nil
	}
	return buildTerminalAppLaunchCommand(scriptPath), nil
}

func writeLaunchScriptTo(w io.Writer, command string) error {
	body := strings.ReplaceAll(launchScriptTemplate, launchCommandPlaceholder, command)
	_, err := io.WriteString(w, body)
	return err
}

func writeLaunchScript(command string) (string, error) {
	f, err := os.CreateTemp("", "apono-launch-*.sh")
	if err != nil {
		return "", err
	}
	if err := writeLaunchScriptTo(f, command); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
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

func buildTerminalAppLaunchCommand(scriptPath string) string {
	return fmt.Sprintf(terminalAppLaunchTemplate, terminalAppName, binZsh, scriptPath, terminalAppName)
}

func buildITermLaunchCommand(scriptPath string) string {
	return fmt.Sprintf(iTermLaunchTemplate, iTermAppName, binZsh, scriptPath, iTermAppName)
}
