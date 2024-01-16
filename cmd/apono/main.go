package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/apono-io/apono-cli/pkg/version"

	"github.com/apono-io/apono-cli/pkg/build"
	"github.com/apono-io/apono-cli/pkg/commands/apono"
)

func main() {
	runner, err := apono.NewRunner(&apono.RunnerOptions{
		VersionInfo: version.VersionInfo{
			BuildDate: build.Date,
			Commit:    build.Commit,
			Version:   build.Version,
		},
	})
	if err != nil {
		fmt.Println("Failed to start CLI: %w", err)
		os.Exit(1)
	}

	err = execute(runner)
	if err != nil {
		fmt.Println("Error:", err.Error())
		fmt.Println("Use '--help' to see usage.")
		os.Exit(1)
	}
}

func execute(runner *apono.Runner) error {
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
