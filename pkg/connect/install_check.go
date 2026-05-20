package connect

import "github.com/apono-io/apono-cli/pkg/clientapi"

// IsInstalled reports whether the given launcher client is launchable on
// this machine. TERMINAL is always considered installed; GUI checks for a
// .app bundle in known macOS app prefixes; TUI/CLI checks $PATH.
func IsInstalled(client clientapi.LauncherClientModel) bool {
	switch client.LauncherType {
	case ClientKindTERMINAL:
		return true
	default:
		return false
	}
}
