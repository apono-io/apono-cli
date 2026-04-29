package actions

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

func Unregister() *cobra.Command {
	return &cobra.Command{
		Use:   "unregister",
		Short: "Remove the apono:// URL handler",
		RunE: func(cmd *cobra.Command, args []string) error {
			if runtime.GOOS != darwinOS {
				return fmt.Errorf("protocol handler is only supported on macOS")
			}

			bundleDir, err := bundlePath()
			if err != nil {
				return err
			}

			if _, statErr := os.Stat(bundleDir); os.IsNotExist(statErr) {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "Protocol handler is not registered")
				return err
			}

			if out, err := exec.Command(lsregisterPath, "-u", bundleDir).CombinedOutput(); err != nil {
				return fmt.Errorf("lsregister -u: %w: %s", err, string(out))
			}

			if err := os.RemoveAll(bundleDir); err != nil {
				return fmt.Errorf("remove %s: %w", bundleDir, err)
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), "Unregistered apono:// handler")
			return err
		},
	}
}
