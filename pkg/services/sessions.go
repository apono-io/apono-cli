package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"slices"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/utils"
)

const (
	CliOutputFormat          = "cli"
	LinkOutputFormat         = "link"
	InstructionsOutputFormat = "instructions"
	JSONOutputFormat         = "json"
	newCredentialsStatus     = "NEW"
)

func ListAccessSessions(ctx context.Context, client *aponoapi.AponoClient, integrationIds []string, bundleIds []string) ([]clientapi.AccessSessionClientModel, error) {
	return utils.GetAllPages(ctx, client, func(ctx context.Context, client *aponoapi.AponoClient, skip int32) ([]clientapi.AccessSessionClientModel, *clientapi.PaginationClientInfoModel, error) {
		listSessionsRequest := client.ClientAPI.AccessSessionsAPI.ListAccessSessions(ctx).Skip(skip)
		if integrationIds != nil {
			listSessionsRequest = listSessionsRequest.IntegrationId(integrationIds)
		}
		if bundleIds != nil {
			listSessionsRequest = listSessionsRequest.BundleId(bundleIds)
		}

		resp, _, err := listSessionsRequest.Execute()
		if err != nil {
			return nil, nil, err
		}

		return resp.Data, &resp.Pagination, nil
	})
}

func ExecuteAccessDetails(cobraCmd *cobra.Command, client *aponoapi.AponoClient, session *clientapi.AccessSessionClientModel) error {
	if runtime.GOOS == "windows" {
		return errors.New("executing cli commands is not supported on windows")
	}

	if !slices.Contains(session.ConnectionMethods, CliOutputFormat) {
		return fmt.Errorf("session %s does not support cli access", session.Id)
	}

	accessDetails, _, err := client.ClientAPI.AccessSessionsAPI.GetAccessSessionAccessDetails(cobraCmd.Context(), session.Id).Execute()
	if err != nil {
		return fmt.Errorf("error getting access details for session id %s: %w", session.Id, err)
	}

	err = executeCommand(cobraCmd, accessDetails.GetCli())
	if err != nil {
		return err
	}

	return nil
}

func executeCommand(cobraCmd *cobra.Command, command string) error {
	if command == "" {
		return errors.New("cannot execute empty command")
	}

	var stderr bytes.Buffer
	cmd := exec.CommandContext(cobraCmd.Context(), "sh", "-c", command)
	cmd.Stdout = cobraCmd.OutOrStdout()
	cmd.Stdin = cobraCmd.InOrStdin()
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error executing command:\n%s\n%s", command, stderr.String())
	}

	return nil
}

func GetSessionDetails(ctx context.Context, client *aponoapi.AponoClient, sessionID string, outputFormat string) (string, error) {
	accessDetails, _, err := client.ClientAPI.AccessSessionsAPI.GetAccessSessionAccessDetails(ctx, sessionID).Execute()
	if err != nil {
		return "", err
	}

	var output string
	switch outputFormat {
	case CliOutputFormat:
		output = *accessDetails.Cli.Get()
	case LinkOutputFormat:
		link := accessDetails.GetLink()
		output = link.GetUrl()
	case InstructionsOutputFormat:
		output = accessDetails.Instructions.Plain
	case JSONOutputFormat:
		var outputBytes []byte
		outputBytes, err = json.Marshal(accessDetails.Json)
		if err != nil {
			return "", err
		}
		output = string(outputBytes)
	}

	return output, nil
}

func IsSessionHaveNewCredentials(session *clientapi.AccessSessionClientModel) bool {
	if session.Credentials.IsSet() {
		credentials := session.Credentials.Get()
		if credentials.Status == newCredentialsStatus && credentials.CanReset {
			return true
		}
	}

	return false
}
