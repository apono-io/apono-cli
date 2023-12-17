package commands

import (
	"fmt"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

func AccessList() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		GroupID: Group.ID,
		Short:   "List all access sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := aponoapi.GetClient(cmd.Context())
			if err != nil {
				return err
			}

			accessSessions, _, err := client.ClientAPI.AccessSessionsAPI.ListAccessSessions(cmd.Context()).Execute()
			if err != nil {
				return err
			}

			table := uitable.New()
			table.AddRow("ID", "NAME", "INTEGRATION NAME", "INTEGRATION TYPE", "TYPE")
			for _, session := range accessSessions.Data {
				table.AddRow(session.Id, session.Name, session.Integration.Name, session.Integration.Type, session.Type.Name)
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), table)
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
