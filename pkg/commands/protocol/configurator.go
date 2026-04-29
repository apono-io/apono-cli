package protocol

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/commands/protocol/actions"
)

type Configurator struct{}

func (c *Configurator) ConfigureCommands(rootCmd *cobra.Command) error {
	protocolCmd := actions.Protocol()
	protocolCmd.AddCommand(actions.Register())
	protocolCmd.AddCommand(actions.Unregister())
	rootCmd.AddCommand(protocolCmd)
	return nil
}
