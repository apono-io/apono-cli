package actions

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

const (
	bundleDirName     = "Apono Connect.app"
	bundleIdentifier  = "io.apono.connect"
	bundleDisplayName = "Apono Connect"
	urlSchemeName     = "Apono Connect"
	urlScheme         = "apono"
)

const lsregisterPath = "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"

// AppleScript shim baked into the .app's Scripts/main.scpt by osacompile.
// The whole script does is forward the URL to handler.sh inside the bundle
// and pop a dialog if it fails. Exit code 64 (EX_USAGE) → invalid URL.
const appleScriptTemplate = `on open location theURL
	set scriptPath to POSIX path of ((path to me as text) & "Contents:Resources:handler.sh")
	try
		do shell script "/bin/zsh -lc " & quoted form of (quoted form of scriptPath & " " & quoted form of theURL)
	on error errMsg number errNum
		if errNum is 64 then
			display dialog "Invalid launch URL. Please try again from the portal." with title "Apono" buttons {"OK"} default button "OK" with icon caution
		else
			display dialog ("Apono failed to launch:" & return & errMsg) with title "Apono" buttons {"OK"} default button "OK" with icon caution
		end if
	end try
end open location
`

// handler.sh body. __APONO_BINARY__ is replaced at register time with the
// absolute path to the running CLI, captured via os.Executable().
const handlerShellTemplate = `#!/bin/zsh
set -e
uri="$1"
if [[ -z "$uri" ]]; then
  echo "missing URI argument" >&2
  exit 64
fi
if [[ "$uri" != apono://connect\?* ]]; then
  echo "unsupported URI: $uri" >&2
  exit 64
fi
query="${uri#*\?}"
session=""; account=""; client=""
for kv in ${(s:&:)query}; do
  case "$kv" in
    session=*) session="${kv#session=}" ;;
    account=*) account="${kv#account=}" ;;
    client=*)  client="${kv#client=}" ;;
  esac
done
if [[ -z "$session" || -z "$account" || -z "$client" ]]; then
  echo "missing required params in: $uri" >&2
  exit 64
fi
# %NN URL-decode (Apono IDs are UUIDs so this rarely fires, but cheap insurance).
session=$(printf '%b' "${session//%/\\x}")
account=$(printf '%b' "${account//%/\\x}")
client=$(printf '%b' "${client//%/\\x}")
export APONO_LAUNCHER_PREVIEW=1
exec "__APONO_BINARY__" access use "$session" --client "$client" --account "$account"
`

func Register() *cobra.Command {
	return &cobra.Command{
		Use:   "register",
		Short: "Register the apono:// URL handler with macOS",
		RunE: func(cmd *cobra.Command, args []string) error {
			if runtime.GOOS != darwinOS {
				return fmt.Errorf("protocol handler is only supported on macOS")
			}

			aponoBinary, err := resolveAponoBinary()
			if err != nil {
				return fmt.Errorf("resolve apono binary: %w", err)
			}

			bundleDir, err := bundlePath()
			if err != nil {
				return err
			}

			if err := buildAppBundle(bundleDir, aponoBinary); err != nil {
				return fmt.Errorf("build app bundle: %w", err)
			}

			if err := registerWithLaunchServices(bundleDir); err != nil {
				return fmt.Errorf("register with LaunchServices: %w", err)
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(),
				"Registered apono:// handler at %s\n", bundleDir)
			return err
		},
	}
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
	apps := filepath.Join(home, "Applications")
	if err := os.MkdirAll(apps, 0o750); err != nil {
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

	if _, err := tmp.WriteString(appleScriptTemplate); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write applescript: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close applescript temp: %w", err)
	}

	out, err := exec.Command("osacompile", "-o", bundleDir, tmpPath).CombinedOutput() //nolint:gosec // bundleDir + tmpPath are constructed internally
	if err != nil {
		return fmt.Errorf("osacompile: %w: %s", err, string(out))
	}
	return nil
}

func patchInfoPlist(bundleDir string) error {
	plist := filepath.Join(bundleDir, "Contents", "Info.plist")
	cmds := [][]string{
		{"write", plist, "CFBundleIdentifier", "-string", bundleIdentifier},
		{"write", plist, "CFBundleName", "-string", bundleDisplayName},
		{"write", plist, "CFBundleDisplayName", "-string", bundleDisplayName},
		{"write", plist, "LSUIElement", "-bool", "true"},
		{"write", plist, "CFBundleURLTypes", "-array",
			fmt.Sprintf(
				`<dict><key>CFBundleURLName</key><string>%s</string>`+
					`<key>CFBundleURLSchemes</key><array><string>%s</string></array></dict>`,
				urlSchemeName, urlScheme,
			)},
	}
	for _, args := range cmds {
		out, err := exec.Command("defaults", args...).CombinedOutput() //nolint:gosec // args constructed from constants
		if err != nil {
			return fmt.Errorf("defaults %s %s: %w: %s", args[0], args[2], err, string(out))
		}
	}
	return nil
}

func writeHandlerScript(bundleDir, aponoBinary string) error {
	body := strings.ReplaceAll(handlerShellTemplate, "__APONO_BINARY__", aponoBinary)
	target := filepath.Join(bundleDir, "Contents", "Resources", "handler.sh")
	if err := os.WriteFile(target, []byte(body), 0o755); err != nil {
		return fmt.Errorf("write handler.sh: %w", err)
	}
	return nil
}

func codesignAdHoc(bundleDir string) error {
	out, err := exec.Command("codesign", "--force", "--sign", "-", bundleDir).CombinedOutput()
	if err != nil {
		return fmt.Errorf("codesign: %w: %s", err, string(out))
	}
	return nil
}

func registerWithLaunchServices(bundleDir string) error {
	out, err := exec.Command(lsregisterPath, "-R", bundleDir).CombinedOutput()
	if err != nil {
		return fmt.Errorf("lsregister: %w: %s", err, string(out))
	}
	return nil
}
