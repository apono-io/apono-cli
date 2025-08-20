package actions

import (
	"fmt"
	"github.com/apono-io/apono-cli/pkg/groups"
	"github.com/spf13/cobra"
)

func MCP() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "mcp",
		Short:             "Run stdio MCP proxy server",
		GroupID:           groups.OtherCommandsGroup.ID,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error { return nil },
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("=== MCP ACK ===")
			return nil
		},
	}

	return cmd
}
