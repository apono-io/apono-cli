package shell

import (
	"context"
	"os/exec"
)

func Command(ctx context.Context, command string) *exec.Cmd {
	shell := "sh"
	if path, err := exec.LookPath("bash"); err == nil && path != "" {
		shell = "bash"
	}
	return exec.CommandContext(ctx, shell, "-c", command)
}
