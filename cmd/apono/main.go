package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/apono-io/apono-cli/pkg/cli"
)

var (
	commit  = "dev"
	date    = time.Now().String()
	version = "0.0.0"
)

func main() {
	runner, err := cli.NewRunner(&cli.RunnerOptions{
		VersionInfo: cli.VersionInfo{
			BuildDate: date,
			Commit:    commit,
			Version:   version,
		},
	})
	if err != nil {
		fmt.Println("Failed to start CLI: %w", err)
		os.Exit(1)
	}

	err = execute(runner)
	if err != nil {
		fmt.Println("Error:", err.Error())
		fmt.Println("See 'apono --help' for usage.")
		os.Exit(1)
	}
}

func execute(runner *cli.Runner) error {
	// trap Ctrl+C and call cancel on the context
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer func() {
		signal.Stop(c)
		cancel()
	}()

	go func() {
		select {
		case <-c:
			cancel()
		case <-ctx.Done():
		}
	}()

	return runner.Run(ctx, os.Args[1:])
}
