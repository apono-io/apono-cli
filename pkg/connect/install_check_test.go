package connect

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/apono-io/apono-cli/pkg/clientapi"
)

func TestIsInstalled_terminalAlwaysTrue(t *testing.T) {
	got := IsInstalled(clientapi.LauncherClientModel{
		Id:           "terminal",
		LauncherType: ClientKindTERMINAL,
	})
	if !got {
		t.Errorf("IsInstalled(TERMINAL) = false, want true")
	}
}

func TestIsInstalled_unknownKindFalse(t *testing.T) {
	got := IsInstalled(clientapi.LauncherClientModel{
		Id:           "weird",
		LauncherType: "WEIRD",
	})
	if got {
		t.Errorf("IsInstalled(unknown kind) = true, want false")
	}
}

func TestIsInstalled_GUI_installedInHomeApplications(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	appDir := filepath.Join(home, "Applications", "Apono-Test-Launcher-That-Does-Not-Exist.app")
	if err := os.MkdirAll(appDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	got := IsInstalled(clientapi.LauncherClientModel{
		Id:           "Apono-Test-Launcher-That-Does-Not-Exist",
		LauncherType: ClientKindGUI,
	})
	if !got {
		t.Errorf("IsInstalled(GUI with bundle in ~/Applications) = false, want true")
	}
}

func TestIsInstalled_GUI_notInstalled(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	got := IsInstalled(clientapi.LauncherClientModel{
		Id:           "Apono-Test-Launcher-That-Does-Not-Exist",
		LauncherType: ClientKindGUI,
	})
	if got {
		t.Errorf("IsInstalled(GUI not on disk) = true, want false")
	}
}

func TestIsInstalled_TUI_onPATH(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "k9s")
	if err := os.Symlink("/bin/sh", binPath); err != nil {
		t.Fatalf("symlink fake binary: %v", err)
	}
	t.Setenv("PATH", dir)

	got := IsInstalled(clientapi.LauncherClientModel{
		Id:           "k9s",
		LauncherType: ClientKindTUI,
	})
	if !got {
		t.Errorf("IsInstalled(TUI on PATH) = false, want true")
	}
}

func TestIsInstalled_TUI_notOnPATH(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	got := IsInstalled(clientapi.LauncherClientModel{
		Id:           "k9s",
		LauncherType: ClientKindTUI,
	})
	if got {
		t.Errorf("IsInstalled(TUI not on PATH) = true, want false")
	}
}

func TestIsInstalled_CLI_onPATH(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "mybin")
	if err := os.Symlink("/bin/sh", binPath); err != nil {
		t.Fatalf("symlink fake binary: %v", err)
	}
	t.Setenv("PATH", dir)

	got := IsInstalled(clientapi.LauncherClientModel{
		Id:           "mybin",
		LauncherType: ClientKindCLI,
	})
	if !got {
		t.Errorf("IsInstalled(CLI on PATH) = false, want true")
	}
}
