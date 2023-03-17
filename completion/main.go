package main

import (
	"log"
	"os"

	"github.com/apono-io/apono-cli/pkg/cli"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s (bash|powershell|zsh)", os.Args[0])
	}

	runner, err := cli.NewRunner(&cli.RunnerOptions{})
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
