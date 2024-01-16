package apono

import (
	"context"
	"fmt"
	"time"

	"github.com/apono-io/apono-cli/pkg/analytics"
	"github.com/apono-io/apono-cli/pkg/version"

	"github.com/apono-io/apono-cli/pkg/commands/access"
	"github.com/apono-io/apono-cli/pkg/commands/auth"
	"github.com/apono-io/apono-cli/pkg/commands/integrations"
	"github.com/apono-io/apono-cli/pkg/commands/requests"
	"github.com/apono-io/apono-cli/pkg/groups"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

func NewRunner(opts *RunnerOptions) (*Runner, error) {
	r := &Runner{
		rootCmd: createRootCommand(opts.VersionInfo),
		opts:    opts,
		configurators: []Configurator{
			&auth.Configurator{},
			&integrations.Configurator{},
			&requests.Configurator{},
			&access.Configurator{},
		},
	}
	err := r.init()
	if err != nil {
		return nil, err
	}

	return r, nil
}

type RunnerOptions struct {
	version.VersionInfo
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

	r.rootCmd.AddGroup(groups.ManagementCommandsGroup)
	r.rootCmd.AddGroup(groups.OtherCommandsGroup)
	r.rootCmd.SetCompletionCommandGroupID(groups.OtherCommandsGroup.ID)
	r.rootCmd.SetHelpCommandGroupID(groups.OtherCommandsGroup.ID)
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

func createRootCommand(versionInfo version.VersionInfo) *cobra.Command {
	c := &cobra.Command{
		Use:           "apono",
		Short:         "View, request and receive permissions to services, DBs and applications directly from your CLI",
		Long:          "Apono Permission Management Automation keeps businesses and their customers moving fast and secure, with simple and precise just in time (JiT) permissions across the RnD stack. You can use this CLI tool to view, request and receive permissions to services, DBs and applications directly",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	c.PersistentFlags().String("profile", "", "profile name")
	c.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		profileName, _ := cmd.Flags().GetString("profile")
		client, err := aponoapi.CreateClient(cmd.Context(), profileName)
		if err != nil {
			return err
		}

		commandStartTime := time.Now()
		commandID := analytics.GenerateCommandID()

		cmd.SetContext(aponoapi.CreateClientContext(cmd.Context(), client))
		cmd.SetContext(version.CreateVersionContext(cmd.Context(), &versionInfo))
		cmd.SetContext(analytics.CreateStartTimeContext(cmd.Context(), &commandStartTime))
		cmd.SetContext(analytics.CreateCommandIDContext(cmd.Context(), commandID))

		return nil
	}
	c.PersistentPostRun = func(cmd *cobra.Command, args []string) {
		analytics.SendCommandAnalyticsEvent(cmd, args)
	}

	return c
}
