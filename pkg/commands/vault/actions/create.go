package actions

import (
	"fmt"

	"github.com/spf13/cobra"
)

func VaultCreate() *cobra.Command {
	var vaultID string
	var value string

	cmd := &cobra.Command{
		Use:   "create <path>",
		Short: "Create a secret in a vault",
		Args:  requirePathArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			vc, secretData, mount, secretName, err := resolveWriteArgs(cmd, args[0], vaultID, value)
			if err != nil {
				return err
			}

			err = vc.WriteSecret(cmd.Context(), mount, secretName, secretData)
			if err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Secret %q created successfully\n", args[0])
			return err
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&vaultID, "vault-id", "", "The vault integration name or ID")
	flags.StringVar(&value, "value", "", "Secret value as JSON (e.g. '{\"key\":\"val\"}')")
	_ = cmd.MarkFlagRequired("vault-id")
	_ = cmd.MarkFlagRequired("value")

	return cmd
}
