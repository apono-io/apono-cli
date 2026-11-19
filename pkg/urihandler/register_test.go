package urihandler

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/apono-io/apono-cli/pkg/utils"
)

func TestHandlerShellTemplate_invokesPATHResolvedApono(t *testing.T) {
	wantSubstrings := []string{
		`exec "$apono_bin" access use`,
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
		"log INFO",          // step logging
		"whence -p apono",   // explicit resolve check before exec
		"trap",              // failure backstop
		"describe_path",     // install-dir reachability captured on failure
		"level=ERROR",       // a malformed launch URL still pages
		"level=WARN",        // per-machine failures do not
		`exec "$apono_bin"`, // still hands off to the CLI
	}
	for _, want := range wantSubstrings {
		if !strings.Contains(handlerShellTemplate, want) {
			t.Errorf("expected handler.sh to contain %q, got:\n%s", want, handlerShellTemplate)
		}
	}
}

func TestHandlerShellTemplate_passesEncodedSessionSafely(t *testing.T) {
	wantSubstrings := []string{
		`export _APONO_SESSION_URL_ENCODED_=1`, // CLI decodes the session exactly once
		`-- "$session"`,                        // session can never be parsed as a flag
		`[[ "$account$client" == *%* ]]`,       // account/client stay unencoded
		`session_has_pct=`,                     // diagnostics for encoded IDs
	}
	for _, want := range wantSubstrings {
		if !strings.Contains(handlerShellTemplate, want) {
			t.Errorf("expected handler.sh to contain %q, got:\n%s", want, handlerShellTemplate)
		}
	}
	unwantedSubstrings := []string{
		`"$session$account$client" == *%*`, // must not reject percent-encoded session IDs (DVL-12422)
		`[[ "$session" == *[^`,             // must not restrict session characters beyond the previous handler
	}
	for _, unwanted := range unwantedSubstrings {
		if strings.Contains(handlerShellTemplate, unwanted) {
			t.Errorf("handler.sh must not contain %q", unwanted)
		}
	}
}

func writeInstalledHandler(t *testing.T, home, body string) {
	t.Helper()
	dir := filepath.Join(home, bundleParentDir, bundleDirName, "Contents", "Resources")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "handler.sh"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRecordBinaryPath_replacesStalePath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	dir := t.TempDir()
	path := filepath.Join(dir, binaryPathFileName)
	if err := os.WriteFile(path, []byte("/old/location/apono\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	recordBinaryPath(dir)
	got, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}
	exe, _ := os.Executable()
	if string(got) != exe+"\n" {
		t.Errorf("recorded %q, want %q", got, exe+"\n")
	}
}

func TestRecordBinaryPath_prefersAponoOnPATH(t *testing.T) {
	binDir := t.TempDir()
	onPath := filepath.Join(binDir, "apono")
	if err := os.WriteFile(onPath, []byte("#!/bin/sh\n"), 0o700); err != nil { //nolint:gosec // fake binary must be executable
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	dir := t.TempDir()
	recordBinaryPath(dir)
	got, err := os.ReadFile(filepath.Clean(filepath.Join(dir, binaryPathFileName)))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != onPath+"\n" {
		t.Errorf("recorded %q, want %q", got, onPath+"\n")
	}
}

func TestUnregister_keepsRecordedPathWhenUnregisterFails(t *testing.T) {
	if runtime.GOOS != utils.DarwinOS {
		t.Skip("darwin only")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, bundleParentDir)
	if err := os.MkdirAll(filepath.Join(dir, bundleDirName), 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, binaryPathFileName)
	if err := os.WriteFile(path, []byte("/x/apono\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Unregister(&bytes.Buffer{}); err == nil {
		t.Fatal("expected lsregister -u to fail for a fake bundle")
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("recorded path must survive a failed unregister, stat err = %v", err)
	}
}

func TestHandlerShellTemplate_readsRecordedPathFromGoLocation(t *testing.T) {
	want := filepath.Join(bundleParentDir, binaryPathFileName)
	if !strings.Contains(handlerShellTemplate, want) {
		t.Errorf("handler.sh must read the recorded path from %q", want)
	}
}

func TestUnregister_removesRecordedPath(t *testing.T) {
	if runtime.GOOS != utils.DarwinOS {
		t.Skip("darwin only")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, bundleParentDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, binaryPathFileName)
	if err := os.WriteFile(path, []byte("/x/apono\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Unregister(&bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected %s to be removed, stat err = %v", path, err)
	}
}

func TestHandlerScriptUpToDate(t *testing.T) {
	t.Run("missing script is stale", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		if handlerScriptUpToDate() {
			t.Error("expected missing handler.sh to be reported stale")
		}
	})
	t.Run("older script is stale", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		writeInstalledHandler(t, home, "#!/bin/zsh\n# older version\n")
		if handlerScriptUpToDate() {
			t.Error("expected outdated handler.sh to be reported stale")
		}
	})
	t.Run("matching script is current", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		writeInstalledHandler(t, home, handlerShellTemplate)
		if !handlerScriptUpToDate() {
			t.Error("expected handler.sh matching the embedded template to be current")
		}
	})
}

func TestRegister_rejectsNonDarwin(t *testing.T) {
	if runtime.GOOS == utils.DarwinOS {
		t.Skip("non-darwin guard test")
	}

	if err := Register(&bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "macOS") {
		t.Errorf("expected non-darwin to error mentioning macOS, got %v", err)
	}
}

func TestReregister_skipsWithoutTerminal(t *testing.T) {
	if err := Reregister(&bytes.Buffer{}); err != nil {
		t.Errorf("expected no error without a terminal, got %v", err)
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
