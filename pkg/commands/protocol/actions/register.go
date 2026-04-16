package actions

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

const (
	appBundleName = "AponoURLHandler.app"
)

// ProtocolRegister creates the "register" command that sets up the apono:// URI scheme handler.
// Currently invoked manually by the user. In the future, this should be called automatically
// as a post-install hook (e.g. Homebrew post_install, deb/rpm postinstall script, Scoop post_install).
func ProtocolRegister() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register apono:// URI scheme handler on macOS",
		RunE: func(cmd *cobra.Command, args []string) error {
			if runtime.GOOS != darwinOS {
				return fmt.Errorf("protocol handler registration is only supported on macOS")
			}

			aponoBinary, err := resolveAponoBinary()
			if err != nil {
				return fmt.Errorf("failed to resolve apono binary path: %w", err)
			}

			appDir, err := appBundlePath()
			if err != nil {
				return err
			}

			err = createAppBundle(appDir, aponoBinary)
			if err != nil {
				return fmt.Errorf("failed to create app bundle: %w", err)
			}

			err = registerWithLaunchServices(appDir)
			if err != nil {
				return fmt.Errorf("failed to register with launch services: %w", err)
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Registered apono:// protocol handler at %s\n", appDir)
			return err
		},
	}

	return cmd
}

func resolveAponoBinary() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}

	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", err
	}

	return exe, nil
}

func appBundlePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	appsDir := filepath.Join(homeDir, "Applications")
	err = os.MkdirAll(appsDir, 0o750)
	if err != nil {
		return "", fmt.Errorf("failed to create ~/Applications: %w", err)
	}

	return filepath.Join(appsDir, appBundleName), nil
}

func createAppBundle(appDir string, aponoBinary string) error {
	// Remove existing bundle to ensure clean state
	_ = os.RemoveAll(appDir)

	// Write the AppleScript that handles the "open location" Apple Event.
	// When macOS opens an apono:// URL, it sends a kAEGetURL event to the app.
	// A plain shell script can't receive Apple Events — only an AppleScript app
	// (or a Cocoa/Swift binary) can. We use osacompile to create a proper .app
	// bundle that has the Apple Event loop built in, then patch its Info.plist
	// to declare the URL scheme.
	// Minimal AppleScript relay — receives the URL from macOS and passes it
	// to `apono protocol handle`. All routing logic lives in Go.
	// The binary path is baked in at registration time so it works regardless
	// of the user's PATH. Re-register after upgrading apono to update the path.
	appleScript := fmt.Sprintf(`on open location theURL
	do shell script "%s protocol handle " & quoted form of theURL & " &"
end open location`, aponoBinary)

	tmpScript, err := os.CreateTemp("", "apono-url-handler-*.applescript")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmpScript.Name()) }()

	_, err = tmpScript.WriteString(appleScript)
	if err != nil {
		return err
	}

	if err = tmpScript.Close(); err != nil {
		return err
	}

	// osacompile -o X.app produces a full .app bundle with Apple Event support
	cmd := exec.Command("osacompile", "-o", appDir, tmpScript.Name()) //nolint:gosec // args are not user-controlled
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to compile AppleScript app: %s: %s", err, string(output))
	}

	// Add URL scheme and app metadata to the existing Info.plist using defaults write.
	// We don't overwrite the plist — osacompile generates keys the AppleScript runtime
	// needs, and replacing the plist breaks the code signature and event handling.
	plistPath := filepath.Join(appDir, "Contents", "Info.plist")

	defaultsCommands := [][]string{
		{"write", plistPath, "CFBundleIdentifier", "-string", "io.apono.url-handler"},
		{"write", plistPath, "CFBundleURLTypes", "-array", `<dict><key>CFBundleURLName</key><string>Apono CLI Protocol</string><key>CFBundleURLSchemes</key><array><string>apono</string></array></dict>`},
		{"write", plistPath, "LSUIElement", "-bool", "true"},
	}

	for _, args := range defaultsCommands {
		out, execErr := exec.Command("defaults", args...).CombinedOutput() //nolint:gosec // args are not user-controlled
		if execErr != nil {
			return fmt.Errorf("defaults write failed: %s: %s", execErr, string(out))
		}
	}

	// Re-sign the app after modifying the plist (ad-hoc signature)
	codesignOut, err := exec.Command("codesign", "--force", "--sign", "-", appDir).CombinedOutput()
	if err != nil {
		return fmt.Errorf("codesign failed: %s: %s", err, string(codesignOut))
	}

	return nil
}

func registerWithLaunchServices(appDir string) error {
	lsregister := "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"
	cmd := exec.Command(lsregister, "-R", appDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}

	return nil
}
