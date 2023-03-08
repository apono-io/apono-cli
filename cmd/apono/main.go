package main

import (
	"github.com/apono-io/apono-cli/pkg/cli"
	"log"
	"os"
	"time"
)

var (
	commit  = "dev"
	date    = time.Now().String()
	version = "0.0.0"
)

func main() {
	runner := cli.NewRunner(&cli.RunnerOptions{
		VersionInfo: cli.VersionInfo{
			BuildDate: date,
			Commit:    commit,
			Version:   version,
		},
	})

	err := runner.Run(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
}
