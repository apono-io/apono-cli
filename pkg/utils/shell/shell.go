package shell

import (
	"context"
	"os/exec"
)

var defaultShell = resolveShell()

func resolveShell() string {
	if _, err := exec.LookPath("bash"); err == nil {
		return "bash"
	}
	return "sh"
}

func Command(ctx context.Context, command string) *exec.Cmd {
	return exec.CommandContext(ctx, defaultShell, "-c", command)
}
