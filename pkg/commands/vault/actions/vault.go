package actions

import (
	"encoding/json"
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/groups"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/spf13/cobra"
)

func Vault() *cobra.Command {
	return &cobra.Command{
		Use:     "vault",
		Short:   "Manage Apono vault secrets",
		GroupID: groups.ManagementCommandsGroup.ID,
	}
}

func requirePathArg(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("requires a secret path argument, e.g.: apono vault %s <mount>/<secret>", cmd.Name())
	}

	if len(args) > 1 {
		return fmt.Errorf("accepts 1 secret path argument but received %d", len(args))
	}

	return nil
}

func resolveWriteArgs(cmd *cobra.Command, secretPath, vaultID, value string) (*services.VaultClient, map[string]interface{}, string, string, error) {
	var secretData map[string]interface{}
	if err := json.Unmarshal([]byte(value), &secretData); err != nil {
		return nil, nil, "", "", fmt.Errorf("invalid JSON value: %w", err)
	}

	client, err := aponoapi.GetClient(cmd.Context())
	if err != nil {
		return nil, nil, "", "", err
	}

	vc, _, err := services.ResolveVaultClient(cmd.Context(), client, vaultID)
	if err != nil {
		return nil, nil, "", "", err
	}

	mount, secretName, err := services.ParseVaultPath(secretPath)
	if err != nil {
		return nil, nil, "", "", err
	}

	return vc, secretData, mount, secretName, nil
}
