package urihandler

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/apono-io/apono-cli/pkg/utils"
)

func TestHandlerShellTemplate_invokesPATHResolvedApono(t *testing.T) {
	wantSubstrings := []string{
		`exec apono access use`,
		`export _APONO_ACCOUNT_ID_="$account"`,
		`--client "$client"`,
		`exit 64`,
	}
	for _, want := range wantSubstrings {
		if !strings.Contains(handlerShellTemplate, want) {
			t.Errorf("expected handler.sh body to contain %q, got:\n%s", want, handlerShellTemplate)
		}
	}
	if strings.Contains(handlerShellTemplate, "__APONO_BINARY__") {
		t.Errorf("handler.sh should not contain __APONO_BINARY__ placeholder (uses PATH-resolved apono now), got:\n%s", handlerShellTemplate)
	}
}

func TestHandlerShellTemplate_writesTraceAndFailureContext(t *testing.T) {
	wantSubstrings := []string{
		filepath.Join(bundleParentDir, handlerLogFileName), // log path agrees with HandlerLogPath
		"log INFO",              // step logging
		"command -v apono",      // explicit resolve check before exec
		"trap",                  // failure backstop
		"PATH=$PATH",            // PATH captured on failure
		"exec apono access use", // still hands off to the CLI
	}
	for _, want := range wantSubstrings {
		if !strings.Contains(handlerShellTemplate, want) {
			t.Errorf("expected handler.sh to contain %q, got:\n%s", want, handlerShellTemplate)
		}
	}
}

// runHandler runs the embedded script under zsh with a controlled HOME and
// PATH, and returns the drained log lines. Skips where zsh is absent (keeps
// non-macOS CI green) — no executable fixture is created, so the gosec file-perm
// rule is never engaged.
func runHandler(t *testing.T, uri, pathEnv string) []LogLine {
	t.Helper()
	zsh, err := exec.LookPath("zsh")
	if err != nil {
		t.Skip("zsh not available")
	}

	home := t.TempDir()
	logDir := filepath.Join(home, bundleParentDir)
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		t.Fatalf("mkdir log dir: %v", err)
	}
	scriptPath := filepath.Join(t.TempDir(), "handler.sh")
	if err := os.WriteFile(scriptPath, []byte(handlerShellTemplate), 0o600); err != nil {
		t.Fatalf("write script: %v", err)
	}

	cmd := exec.Command(zsh, scriptPath, uri)
	cmd.Env = []string{"HOME=" + home, "PATH=" + pathEnv}
	_ = cmd.Run() // non-zero exit is expected on the failure paths

	lines, err := DrainLog(filepath.Join(logDir, handlerLogFileName))
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	return lines
}

func TestHandlerShell_badURI_logsInvalidLaunchURL(t *testing.T) {
	if runtime.GOOS != utils.DarwinOS {
		// The script logic is portable, but run it only where launches actually
		// happen to avoid depending on a Linux zsh being present/configured.
		t.Skip("handler execution test runs on macOS")
	}
	lines := runHandler(t, "apono://wrong", "/usr/bin:/bin")

	if !containsMessage(lines, "invalid launch URL") || !containsMessage(lines, "code=64") {
		t.Errorf("expected an ERROR line for the bad URI, got %+v", lines)
	}
}

func TestHandlerShell_aponoMissing_logsNotFoundWithPATH(t *testing.T) {
	if runtime.GOOS != utils.DarwinOS {
		t.Skip("handler execution test runs on macOS")
	}
	// Valid URI, but a PATH deliberately without apono.
	lines := runHandler(t, "apono://connect?session=s&account=a&client=dbeaver", "/usr/bin:/bin")

	if !containsMessage(lines, "received launch request") {
		t.Errorf("expected the initial INFO step, got %+v", lines)
	}
	if !containsMessage(lines, "apono CLI not found") || !containsMessage(lines, "code=127") {
		t.Errorf("expected an ERROR line for missing apono, got %+v", lines)
	}
}

func containsMessage(lines []LogLine, substr string) bool {
	for _, l := range lines {
		if strings.Contains(l.Message, substr) {
			return true
		}
	}
	return false
}

func TestRegister_rejectsNonDarwin(t *testing.T) {
	if runtime.GOOS == utils.DarwinOS {
		t.Skip("non-darwin guard test")
	}

	if err := Register(&bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "macOS") {
		t.Errorf("expected non-darwin to error mentioning macOS, got %v", err)
	}
}

func TestUnregister_rejectsNonDarwin(t *testing.T) {
	if runtime.GOOS == utils.DarwinOS {
		t.Skip("non-darwin guard test")
	}

	if err := Unregister(&bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "macOS") {
		t.Errorf("expected non-darwin to error mentioning macOS, got %v", err)
	}
}
