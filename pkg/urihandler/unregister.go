package urihandler

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
)

func Unregister(out io.Writer) error {
	if runtime.GOOS != darwinOS {
		return fmt.Errorf("access-handler is only supported on macOS")
	}

	bundleDir, err := bundlePath()
	if err != nil {
		return err
	}

	if _, statErr := os.Stat(bundleDir); os.IsNotExist(statErr) {
		_, err = fmt.Fprintln(out, "Protocol handler is not registered")
		return err
	}

	if lsOut, lsErr := unregisterFromLaunchServices(bundleDir); lsErr != nil {
		return fmt.Errorf("lsregister -u: %w: %s", lsErr, string(lsOut))
	}

	if err = os.RemoveAll(bundleDir); err != nil {
		return fmt.Errorf("remove %s: %w", bundleDir, err)
	}

	_, err = fmt.Fprintln(out, "Unregistered apono:// handler")
	return err
}

func unregisterFromLaunchServices(bundleDir string) ([]byte, error) {
	return exec.CommandContext(context.Background(), lsregisterPath, "-u", bundleDir).CombinedOutput()
}
