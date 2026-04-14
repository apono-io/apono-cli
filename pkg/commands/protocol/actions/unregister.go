package actions

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

// ProtocolUnregister creates the "unregister" command that removes the apono:// URI scheme handler.
// In the future, this should be called automatically as a pre-uninstall hook during CLI removal.
func ProtocolUnregister() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unregister",
		Short: "Remove apono:// URI scheme handler from macOS",
		RunE: func(cmd *cobra.Command, args []string) error {
			if runtime.GOOS != "darwin" {
				return fmt.Errorf("protocol handler is only supported on macOS")
			}

			appDir, err := appBundlePath()
			if err != nil {
				return err
			}

			if _, err := os.Stat(appDir); os.IsNotExist(err) {
				fmt.Fprintln(cmd.OutOrStdout(), "Protocol handler is not registered")
				return nil
			}

			lsregister := "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"
			unregCmd := exec.Command(lsregister, "-u", appDir)
			output, err := unregCmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to unregister: %s: %s", err, string(output))
			}

			err = os.RemoveAll(appDir)
			if err != nil {
				return fmt.Errorf("failed to remove app bundle: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Unregistered apono:// protocol handler")
			return nil
		},
	}

	return cmd
}
