package actions

import (
	"strings"
	"testing"

	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/connect"
)

func TestNoInstalledClientsError_emptyClientList(t *testing.T) {
	err := noInstalledClientsError(nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no GUI or TUI launchers") {
		t.Errorf("expected error to mention no GUI/TUI launchers, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "--run") {
		t.Errorf("expected error to point at --run as fallback, got %q", err.Error())
	}
}

func TestNoInstalledClientsError_onlyTerminalLauncherInList(t *testing.T) {
	// Terminal launchers are excluded from the suggestion list — same error
	// shape as an empty list.
	clients := []clientapi.LauncherClientModel{
		{Id: "terminal", LauncherType: connect.ClientKindTERMINAL, DisplayName: "Open in Terminal"},
	}
	err := noInstalledClientsError(clients)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no GUI or TUI launchers") {
		t.Errorf("expected error to mention no GUI/TUI launchers, got %q", err.Error())
	}
}

func TestNoInstalledClientsError_listsGUIAndTUINames(t *testing.T) {
	clients := []clientapi.LauncherClientModel{
		{Id: "dbeaver", LauncherType: connect.ClientKindGUI, DisplayName: "DBeaver"},
		{Id: "k9s", LauncherType: connect.ClientKindTUI, DisplayName: "k9s"},
		{Id: "terminal", LauncherType: connect.ClientKindTERMINAL, DisplayName: "Open in Terminal"},
	}
	err := noInstalledClientsError(clients)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Install a CLI tool") {
		t.Errorf("expected 'Install a CLI tool' phrasing, got %q", err.Error())
	}
	for _, want := range []string{"DBeaver", "k9s"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected %q in suggestion list, got %q", want, err.Error())
		}
	}
	if strings.Contains(err.Error(), "Open in Terminal") {
		t.Errorf("terminal launcher should not appear in installable list, got %q", err.Error())
	}
}
