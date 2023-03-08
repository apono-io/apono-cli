package cli

import (
	"github.com/spf13/cobra"
)

func NewRunner(opts *RunnerOptions) *Runner {
	r := &Runner{
		rootCmd: &cobra.Command{
			Use:           "apono",
			Short:         "Oneliner description about apono CLI",
			Long:          "More in dept description about apono CLI",
			SilenceErrors: true,
			SilenceUsage:  true,
		},
		opts: opts,
	}
	r.init()
	return r
}

type RunnerOptions struct {
	VersionInfo
}

type Runner struct {
	rootCmd *cobra.Command
	opts    *RunnerOptions
}

func (r *Runner) Run(args []string) error {
	r.rootCmd.SetArgs(args)
	return r.rootCmd.Execute()
}

func (r *Runner) init() {
	r.rootCmd.AddCommand(VersionCommand(r.opts.VersionInfo))
}
