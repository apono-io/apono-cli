package connect

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/apono-io/apono-cli/pkg/clientapi"
)

const (
	appBundleExt   = ".app"
	systemAppsDir  = "/Applications"
	setappAppsDir  = "/Applications/Setapp"
	userAppsSubdir = "Applications"
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
	case ClientKindTUI, ClientKindCLI:
		_, err := exec.LookPath(client.Id)
		return err == nil
	default:
		return false
	}
}

func guiBundleExists(id string) bool {
	home, _ := os.UserHomeDir()
	bundle := id + appBundleExt
	candidates := []string{
		filepath.Join(systemAppsDir, bundle),
		filepath.Join(home, userAppsSubdir, bundle),
		filepath.Join(setappAppsDir, bundle),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}
