package urihandler

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

//go:embed handler.applescript
var appleScriptTemplate string

//go:embed handler.sh
var handlerShellTemplate string

const darwinOS = "darwin"

const (
	bundleDirName     = "Apono Connect.app"
	bundleIdentifier  = "io.apono.connect"
	bundleDisplayName = "Apono Connect"
	urlSchemeName     = "Apono Connect"
	urlScheme         = "apono"

	applicationsDirName             = "Applications"
	applicationsDirPerm os.FileMode = 0o700
	handlerScriptPerm   os.FileMode = 0o600
)

const lsregisterPath = "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"

const urlTypesValue = `<dict><key>CFBundleURLName</key><string>` + urlSchemeName +
	`</string><key>CFBundleURLSchemes</key><array><string>` +
	urlScheme +
	`</string></array></dict>`

var infoPlistEntries = []struct {
	key, valueType, value string
}{
	{"CFBundleIdentifier", "-string", bundleIdentifier},
	{"CFBundleName", "-string", bundleDisplayName},
	{"CFBundleDisplayName", "-string", bundleDisplayName},
	{"LSUIElement", "-bool", "true"},
	{"CFBundleURLTypes", "-array", urlTypesValue},
}

// Register currently builds the bundle on user invocation. Long-term this
// becomes a package-install hook (Homebrew post-install, deb/rpm postinst,
// Scoop) - manual for now.
func Register(out io.Writer) error {
	if runtime.GOOS != darwinOS {
		return fmt.Errorf("access-handler is only supported on macOS")
	}

	aponoBinary, err := resolveAponoBinary()
	if err != nil {
		return fmt.Errorf("resolve apono binary: %w", err)
	}

	bundleDir, err := bundlePath()
	if err != nil {
		return err
	}

	if err = buildAppBundle(bundleDir, aponoBinary); err != nil {
		return fmt.Errorf("build app bundle: %w", err)
	}

	if err = registerWithLaunchServices(bundleDir); err != nil {
		return fmt.Errorf("register with LaunchServices: %w", err)
	}

	_, err = fmt.Fprintf(out, "Registered apono:// handler at %s\n", bundleDir)
	return err
}

func resolveAponoBinary() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

func bundlePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	apps := filepath.Join(home, applicationsDirName)
	if err = os.MkdirAll(apps, applicationsDirPerm); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", apps, err)
	}
	return filepath.Join(apps, bundleDirName), nil
}

func buildAppBundle(bundleDir, aponoBinary string) error {
	// Wipe any prior bundle for clean state — register is idempotent.
	if err := os.RemoveAll(bundleDir); err != nil {
		return fmt.Errorf("clean previous bundle: %w", err)
	}

	if err := compileAppleScript(bundleDir); err != nil {
		return err
	}
	if err := patchInfoPlist(bundleDir); err != nil {
		return err
	}
	if err := writeHandlerScript(bundleDir, aponoBinary); err != nil {
		return err
	}
	// codesign breaks after Info.plist edits; re-sign ad-hoc so LaunchServices
	// is willing to dispatch Apple Events to the bundle.
	return codesignAdHoc(bundleDir)
}

func compileAppleScript(bundleDir string) error {
	tmp, err := os.CreateTemp("", "apono-handler-*.applescript")
	if err != nil {
		return fmt.Errorf("temp applescript: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err = tmp.WriteString(appleScriptTemplate); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write applescript: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("close applescript temp: %w", err)
	}

	out, err := runOsaCompile(bundleDir, tmpPath)
	if err != nil {
		return fmt.Errorf("osacompile: %w: %s", err, string(out))
	}
	return nil
}

func runOsaCompile(outputBundle, scriptPath string) ([]byte, error) {
	return exec.CommandContext(context.Background(), "osacompile", "-o", outputBundle, scriptPath).CombinedOutput()
}

func patchInfoPlist(bundleDir string) error {
	plist := filepath.Join(bundleDir, "Contents", "Info.plist")
	for _, e := range infoPlistEntries {
		if err := writePlistEntry(plist, e.key, e.valueType, e.value); err != nil {
			return err
		}
	}
	return nil
}

func writePlistEntry(plist, key, valueType, value string) error {
	out, err := exec.CommandContext(context.Background(), "defaults", "write", plist, key, valueType, value).CombinedOutput()
	if err != nil {
		return fmt.Errorf("defaults write %s: %w: %s", key, err, string(out))
	}
	return nil
}

// handlerScriptBody returns the handler.sh contents with the apono binary path
// substituted. Pure, side-effect-free — used by writeHandlerScript and tests.
func handlerScriptBody(aponoBinary string) string {
	return strings.ReplaceAll(handlerShellTemplate, "__APONO_BINARY__", aponoBinary)
}

func writeHandlerScript(bundleDir, aponoBinary string) error {
	target := filepath.Join(bundleDir, "Contents", "Resources", "handler.sh")
	// AppleScript invokes this via `zsh -l handler.sh ...`, so the file is
	// read by zsh as a script — no executable bit required.
	if err := os.WriteFile(target, []byte(handlerScriptBody(aponoBinary)), handlerScriptPerm); err != nil {
		return fmt.Errorf("write handler.sh: %w", err)
	}
	return nil
}

func codesignAdHoc(bundleDir string) error {
	out, err := exec.CommandContext(context.Background(), "codesign", "--force", "--sign", "-", bundleDir).CombinedOutput()
	if err != nil {
		return fmt.Errorf("codesign: %w: %s", err, string(out))
	}
	return nil
}

func registerWithLaunchServices(bundleDir string) error {
	out, err := exec.CommandContext(context.Background(), lsregisterPath, "-R", bundleDir).CombinedOutput()
	if err != nil {
		return fmt.Errorf("lsregister: %w: %s", err, string(out))
	}
	return nil
}
