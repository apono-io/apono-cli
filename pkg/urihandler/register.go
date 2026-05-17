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
)

//go:embed scripts/handler.applescript
var appleScriptTemplate string

//go:embed scripts/handler.sh
var handlerShellTemplate string

const darwinOS = "darwin"

const (
	bundleDirName     = "Apono Connect.app"
	bundleIdentifier  = "io.apono.connect"
	bundleDisplayName = "Apono Connect"
	urlSchemeName     = "Apono Connect"
	urlScheme         = "apono"

	bundleParentDir                 = "Library/Application Support/apono-cli"
	legacyBundleDir                 = "Applications"
	bundleParentDirPerm os.FileMode = 0o700
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

// Register builds the apono:// handler bundle at the canonical location and
// registers it with LaunchServices. Used as a manual fallback (e.g. for
// non-brew installs) and by the login-time auto-register flow. Brew installs
// don't call this — they ship a pre-built bundle and use `open -a` to launch
// it (see scripts/handler.applescript's `on run` self-install handler).
func Register(out io.Writer) error {
	if runtime.GOOS != darwinOS {
		return fmt.Errorf("access-handler is only supported on macOS")
	}

	bundleDir, err := bundlePath()
	if err != nil {
		return err
	}

	if err = BuildBundleAt(bundleDir); err != nil {
		return fmt.Errorf("build app bundle: %w", err)
	}

	unregisterLegacyBundle()

	if err = registerWithLaunchServices(bundleDir); err != nil {
		return fmt.Errorf("register with LaunchServices: %w", err)
	}

	_, err = fmt.Fprintf(out, "Registered apono:// handler at %s\n", bundleDir)
	return err
}

// BuildBundleAt produces a complete apono:// handler bundle at the given path.
// Used at release time by goreleaser to pre-build the bundle for the brew
// tarball, and at runtime by Register for the manual / login-time fallback.
func BuildBundleAt(bundleDir string) error {
	if err := os.RemoveAll(bundleDir); err != nil {
		return fmt.Errorf("clean previous bundle: %w", err)
	}
	if err := compileAppleScript(bundleDir); err != nil {
		return err
	}
	if err := patchInfoPlist(bundleDir); err != nil {
		return err
	}
	if err := writeHandlerScript(bundleDir); err != nil {
		return err
	}
	return codesignAdHoc(bundleDir)
}

func bundlePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	parent := filepath.Join(home, bundleParentDir)
	if err = os.MkdirAll(parent, bundleParentDirPerm); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", parent, err)
	}
	return filepath.Join(parent, bundleDirName), nil
}

func unregisterLegacyBundle() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	legacyPath := filepath.Join(home, legacyBundleDir, bundleDirName)
	if _, err := os.Stat(legacyPath); err != nil {
		return
	}
	_, _ = unregisterFromLaunchServices(legacyPath)
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

func writeHandlerScript(bundleDir string) error {
	target := filepath.Join(bundleDir, "Contents", "Resources", "handler.sh")
	// AppleScript invokes this via `zsh -l handler.sh ...`, so the file is
	// read by zsh as a script — no executable bit required.
	if err := os.WriteFile(target, []byte(handlerShellTemplate), handlerScriptPerm); err != nil {
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
