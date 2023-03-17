package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra/doc"

	"github.com/apono-io/apono-cli/pkg/requests"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/auth"
)

func NewRunner(opts *RunnerOptions) (*Runner, error) {
	r := &Runner{
		rootCmd: &cobra.Command{
			Use:           "apono",
			Short:         "Oneliner description about apono CLI",
			Long:          "More in dept description about apono CLI",
			SilenceErrors: true,
			SilenceUsage:  true,
		},
		opts: opts,
		configurators: []Configurator{
			&auth.Configurator{},
			&requests.Configurator{},
		},
	}
	err := r.init()
	if err != nil {
		return nil, err
	}

	return r, nil
}

type RunnerOptions struct {
	VersionInfo
}

type Runner struct {
	rootCmd       *cobra.Command
	opts          *RunnerOptions
	configurators []Configurator
}

func (r *Runner) Run(ctx context.Context, args []string) error {
	r.rootCmd.SetArgs(args)
	return r.rootCmd.ExecuteContext(ctx)
}

func (r *Runner) init() error {
	for _, configurator := range r.configurators {
		err := configurator.ConfigureCommands(r.rootCmd)
		if err != nil {
			return fmt.Errorf("failed to configure commands: %w", err)
		}
	}

	r.rootCmd.AddGroup(otherCommandsGroup)
	r.rootCmd.SetCompletionCommandGroupID(otherCommandsGroup.ID)
	r.rootCmd.SetHelpCommandGroupID(otherCommandsGroup.ID)
	r.rootCmd.AddCommand(VersionCommand(r.opts.VersionInfo))

	return nil
}

func (r *Runner) GenBashCompletionFile(filename string) error {
	return r.rootCmd.GenBashCompletionFile(filename)
}

func (r *Runner) GenPowerShellCompletionFile(filename string) error {
	return r.rootCmd.GenPowerShellCompletionFile(filename)
}

func (r *Runner) GenZshCompletionFile(filename string) error {
	return r.rootCmd.GenZshCompletionFile(filename)
}

func (r *Runner) GenManTree(dir string) error {
	header := &doc.GenManHeader{
		Title:   "apono",
		Section: "1",
	}

	return doc.GenManTree(r.rootCmd, header, dir)
}

var otherCommandsGroup = &cobra.Group{
	ID:    "other",
	Title: "Other Commands",
}
