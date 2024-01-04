package main

import (
	"log"

	"github.com/apono-io/apono-cli/pkg/commands/apono"
)

func main() {
	runner, err := apono.NewRunner(&apono.RunnerOptions{})
	if err != nil {
		log.Fatal(err)
	}

	err = runner.GenManTree("./contrib/manpage")
	if err != nil {
		log.Fatal(err)
	}
}
