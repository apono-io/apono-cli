package shell

import (
	"os/exec"
)

var DefaultShell = resolveShell()

func resolveShell() string {
	if _, err := exec.LookPath("bash"); err == nil {
		return "bash"
	}
	return "sh"
}
