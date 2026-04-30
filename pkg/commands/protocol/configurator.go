package protocol

import (
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/commands/protocol/actions"
)

type Configurator struct{}

func (c *Configurator) ConfigureCommands(rootCmd *cobra.Command) error {
	protocolCmd := actions.Protocol()
	rootCmd.AddCommand(protocolCmd)

	protocolCmd.AddCommand(actions.ProtocolRegister())
	protocolCmd.AddCommand(actions.ProtocolHandle())
	protocolCmd.AddCommand(actions.ProtocolUnregister())
	return nil
}
