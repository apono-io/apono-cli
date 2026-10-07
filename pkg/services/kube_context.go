package services

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"

	"github.com/gookit/color"
	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
)

// contextNamePattern matches a kubectl context name, optionally quoted. Names containing
// whitespace are not supported: a quoted name with a space is truncated at the space.
// EKS, GKE and AKS context names never contain whitespace.
const contextNamePattern = `['"]?([^\s'"&;|]+)`

var (
	useContextPattern = regexp.MustCompile(`kubectl\s+config\s+use-context\s+` + contextNamePattern)
	setContextPattern = regexp.MustCompile(`kubectl\s+config\s+set-context\s+` + contextNamePattern)
)

type kubeCommandResolver struct {
	contextExists    func(ctx context.Context, name string) (bool, error)
	resetCredentials func(ctx context.Context, client *aponoapi.AponoClient, sessionID string) error
	fetchCliCommand  func(ctx context.Context, client *aponoapi.AponoClient, sessionID string) (string, error)
}

var defaultKubeCommandResolver = kubeCommandResolver{
	contextExists:    kubectlContextExists,
	resetCredentials: ResetSessionCredentials,
	fetchCliCommand:  fetchCliCommand,
}

// resolveKubeCliCommand makes sure a kubectl use-context command has a context to switch to.
// When credentials were already consumed, the backend returns only `kubectl config use-context`;
// if that context is missing from the kubeconfig, new credentials are issued so the full
// set-cluster/set-credentials/set-context command can be run instead.
func resolveKubeCliCommand(cobraCmd *cobra.Command, client *aponoapi.AponoClient, session *clientapi.AccessSessionClientModel, command string) (string, error) {
	return defaultKubeCommandResolver.resolve(cobraCmd, client, session, command)
}

func (r kubeCommandResolver) resolve(cobraCmd *cobra.Command, client *aponoapi.AponoClient, session *clientapi.AccessSessionClientModel, command string) (string, error) {
	contextName, ok := extractKubeContext(command)
	if !ok || setsKubeContext(command, contextName) {
		return command, nil
	}

	ctx := cobraCmd.Context()
	exists, err := r.contextExists(ctx, contextName)
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

	err = r.resetCredentials(ctx, client, session.Id)
	if err != nil {
		return "", fmt.Errorf("failed to create kubernetes context %q: %w", contextName, err)
	}

	newCommand, err := r.fetchCliCommand(ctx, client, session.Id)
	if err != nil {
		return "", fmt.Errorf("failed to create kubernetes context %q: %w", contextName, err)
	}

	if !setsKubeContext(newCommand, contextName) {
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

// setsKubeContext reports whether the command creates the named context itself,
// so it does not need to exist in the kubeconfig beforehand.
func setsKubeContext(command, name string) bool {
	for _, match := range setContextPattern.FindAllStringSubmatch(command, -1) {
		if match[1] == name {
			return true
		}
	}

	return false
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
