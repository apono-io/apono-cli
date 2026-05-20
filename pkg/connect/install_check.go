package connect

import (
	"os"
	"path/filepath"

	"github.com/apono-io/apono-cli/pkg/clientapi"
)

// IsInstalled reports whether the given launcher client is launchable on
// this machine. TERMINAL is always considered installed; GUI checks for a
// .app bundle in known macOS app prefixes; TUI/CLI checks $PATH.
func IsInstalled(client clientapi.LauncherClientModel) bool {
	switch client.LauncherType {
	case ClientKindTERMINAL:
		return true
	case ClientKindGUI:
		return guiBundleExists(client.Id)
	default:
		return false
	}
}

func guiBundleExists(id string) bool {
	home, _ := os.UserHomeDir()
	candidates := []string{
		"/Applications/" + id + ".app",
		filepath.Join(home, "Applications", id+".app"),
		"/Applications/Setapp/" + id + ".app",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}
