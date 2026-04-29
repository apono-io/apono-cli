package actions

import (
	"fmt"
	"os"
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
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "Protocol handler is not registered")
				return err
			}

			if out, lsErr := runCommand(lsregisterPath, "-u", bundleDir).CombinedOutput(); lsErr != nil {
				return fmt.Errorf("lsregister -u: %w: %s", lsErr, string(out))
			}

			if err = os.RemoveAll(bundleDir); err != nil {
				return fmt.Errorf("remove %s: %w", bundleDir, err)
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), "Unregistered apono:// handler")
			return err
		},
	}
}
