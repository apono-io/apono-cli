package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/utils"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
)

const (
	CliOutputFormat          = "cli"
	LinkOutputFormat         = "link"
	InstructionsOutputFormat = "instructions"
	JSONOutputFormat         = "json"
	newCredentialsStatus     = "NEW"
)

func PrintAccessSessionDetails(cmd *cobra.Command, sessions []clientapi.AccessSessionClientModel, format *utils.Format) error {
	switch *format {
	case utils.TableFormat:
		table := generateSessionsTable(sessions)

		_, err := fmt.Fprintln(cmd.OutOrStdout(), table)
		return err
	case utils.JSONFormat:
		return utils.PrintObjectsAsJSON(cmd.OutOrStdout(), sessions)
	case utils.YamlFormat:
		return utils.PrintObjectsAsYaml(cmd.OutOrStdout(), sessions)
	default:
		return fmt.Errorf("unsupported output format")
	}
}

func generateSessionsTable(sessions []clientapi.AccessSessionClientModel) *uitable.Table {
	table := uitable.New()
	table.AddRow("ID", "NAME", "INTEGRATION NAME", "INTEGRATION TYPE", "TYPE")
	for _, session := range sessions {
		table.AddRow(session.Id, session.Name, session.Integration.Name, session.Integration.Type, session.Type.Name)
	}

	return table
}

func ListAccessSessions(ctx context.Context, client *aponoapi.AponoClient, integrationIds []string, bundleIds []string, requestIds []string) ([]clientapi.AccessSessionClientModel, error) {
	return utils.GetAllPages(ctx, client, func(ctx context.Context, client *aponoapi.AponoClient, skip int32) ([]clientapi.AccessSessionClientModel, *clientapi.PaginationClientInfoModel, error) {
		listSessionsRequest := client.ClientAPI.AccessSessionsAPI.ListAccessSessions(ctx).Skip(skip)
		if integrationIds != nil {
			listSessionsRequest = listSessionsRequest.IntegrationId(integrationIds)
		}
		if bundleIds != nil {
			listSessionsRequest = listSessionsRequest.BundleId(bundleIds)
		}
		if requestIds != nil {
			listSessionsRequest = listSessionsRequest.RequestId(requestIds)
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

	if !utils.Contains(session.ConnectionMethods, CliOutputFormat) {
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
