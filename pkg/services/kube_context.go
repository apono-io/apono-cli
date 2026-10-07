package services

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/gookit/color"
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
)

const kubeSetContextCommand = "kubectl config set-context"

var useContextPattern = regexp.MustCompile(`kubectl\s+config\s+use-context\s+['"]?([^\s'"&;|]+)`)

var (
	kubeContextExists  = kubectlContextExists
	resetCredentialsFn = ResetSessionCredentials
	fetchCliCommandFn  = fetchCliCommand
)

// resolveKubeCliCommand makes sure a kubectl use-context command has a context to switch to.
// When credentials were already consumed, the backend returns only `kubectl config use-context`;
// if that context is missing from the kubeconfig, new credentials are issued so the full
// set-cluster/set-credentials/set-context command can be run instead.
func resolveKubeCliCommand(cobraCmd *cobra.Command, client *aponoapi.AponoClient, session *clientapi.AccessSessionClientModel, command string) (string, error) {
	contextName, ok := extractKubeContext(command)
	if !ok || strings.Contains(command, kubeSetContextCommand) {
		return command, nil
	}

	ctx := cobraCmd.Context()
	exists, err := kubeContextExists(ctx, contextName)
	if err != nil || exists {
		return command, nil
	}

	if !session.Credentials.IsSet() || !session.Credentials.Get().CanReset {
		return "", fmt.Errorf("kubernetes context %q does not exist in your kubeconfig and the credentials for session %s cannot be reissued. request access again to connect", contextName, session.Id)
	}

	_, err = fmt.Fprintf(cobraCmd.OutOrStdout(), "\nKubernetes context %s not found in your kubeconfig, creating it...\n\n", color.Green.Sprint(contextName))
	if err != nil {
		return "", err
	}

	err = resetCredentialsFn(ctx, client, session.Id)
	if err != nil {
		return "", fmt.Errorf("failed to create kubernetes context %q: %w", contextName, err)
	}

	newCommand, err := fetchCliCommandFn(ctx, client, session.Id)
	if err != nil {
		return "", fmt.Errorf("failed to create kubernetes context %q: %w", contextName, err)
	}

	if !strings.Contains(newCommand, kubeSetContextCommand) {
		return "", fmt.Errorf("failed to create kubernetes context %q: new credentials were not returned for session %s", contextName, session.Id)
	}

	return newCommand, nil
}

func extractKubeContext(command string) (string, bool) {
	match := useContextPattern.FindStringSubmatch(command)
	if match == nil {
		return "", false
	}

	return match[1], true
}

func kubectlContextExists(ctx context.Context, name string) (bool, error) {
	err := exec.CommandContext(ctx, "kubectl", "config", "get-contexts", name, "-o", "name").Run()
	if err == nil {
		return true, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return false, nil
	}

	return false, err
}

func fetchCliCommand(ctx context.Context, client *aponoapi.AponoClient, sessionID string) (string, error) {
	accessDetails, _, err := client.ClientAPI.AccessSessionsAPI.GetAccessSessionAccessDetails(ctx, sessionID).
		ConsumedBy(aponoapi.ConsumedByAponoCli).
		Execute()
	if err != nil {
		return "", err
	}

	return accessDetails.GetCli(), nil
}
