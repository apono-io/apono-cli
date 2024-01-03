package main

import (
	"log"
	"os"

	"github.com/apono-io/apono-cli/pkg/commands/apono"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s (bash|powershell|zsh)", os.Args[0])
	}

	runner, err := apono.NewRunner(&apono.RunnerOptions{})
	if err != nil {
		log.Fatal(err)
	}

	shell := os.Args[1]
	switch shell {
	case "bash":
		err = runner.GenBashCompletionFile("bash_completion")
	case "powershell":
		err = runner.GenPowerShellCompletionFile("powershell_completion")
	case "zsh":
		err = runner.GenZshCompletionFile("zsh_completion")
	default:
		log.Fatalf("unsupported shell %q", shell)
	}

	if err != nil {
		log.Fatal(err)
	}
}
