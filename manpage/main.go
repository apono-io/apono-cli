package main

import (
	"log"

	"github.com/apono-io/apono-cli/pkg/cli"
)

func main() {
	runner, err := cli.NewRunner(&cli.RunnerOptions{})
	if err != nil {
		log.Fatal(err)
	}

	err = runner.GenManTree("./contrib/manpage")
	if err != nil {
		log.Fatal(err)
	}
}
