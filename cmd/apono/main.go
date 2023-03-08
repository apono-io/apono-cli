package main

import (
	"fmt"
	"os"
	"time"

	"github.com/apono-io/apono-cli/pkg/cli"
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
		fmt.Println("Error:", err.Error())
		fmt.Println("See 'apono --help' for usage.")
		os.Exit(1)
	}
}
