package flows

import (
	"fmt"
	"runtime"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/connect"
	"github.com/apono-io/apono-cli/pkg/interactive/selectors"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/styles"
	"github.com/apono-io/apono-cli/pkg/utils"

	"github.com/gookit/color"
	"github.com/spf13/cobra"
)

const (
	resetCredentialsCommand = "apono access reset-credentials "
)

func RunUseSessionInteractiveFlow(cmd *cobra.Command, client *aponoapi.AponoClient, requestIDFilter string) error {
	session, err := selectors.RunSessionsSelector(cmd.Context(), client, requestIDFilter)
	if err != nil {
		return err
	}

	if len(session.ConnectionMethods) == 0 {
		return fmt.Errorf("no connection methods found for session %s", session.Id)
	}

	if len(session.ConnectionMethods) == 1 {
		err = printSessionInstructions(cmd, client, session)
		if err != nil {
			return err
		}

		return nil
	}

	connectWithAppAvailable := false
	var guiTuiInstalled []clientapi.LauncherClientModel
	if runtime.GOOS == utils.DarwinOS {

		if result, fetchErr := connect.FetchClients(cmd.Context(), client, session.Id); fetchErr == nil {
			for _, c := range result.Clients {
				if (c.LauncherType == connect.ClientKindGUI || c.LauncherType == connect.ClientKindTUI) && connect.IsInstalled(c) {
					guiTuiInstalled = append(guiTuiInstalled, c)
				}
			}
		}
		connectWithAppAvailable = len(guiTuiInstalled) > 0
	}

	accessMethod, err := selectors.RunSessionCliMethodOptionSelector(connectWithAppAvailable)
	if err != nil {
		return err
	}

	switch accessMethod {
	case selectors.ExecuteOption:
		err = PrintErrorConnectingSuggestion(cmd, session.Id)
		if err != nil {
			return err
		}

		err = services.ExecuteAccessDetails(cmd, client, session)
		return err

	case selectors.PrintOption:
		err = printSessionInstructions(cmd, client, session)
		return err

	case selectors.ExectueWithAppOption:
		selectedID, err := selectors.RunLauncherClientSelector(guiTuiInstalled)
		if err != nil {
			return err
		}
		if err := PrintErrorConnectingSuggestion(cmd, session.Id); err != nil {
			return err
		}
		return connect.NewClientStarter().Start(cmd, client, session.Id, selectedID)

	default:
		return fmt.Errorf("unknown access method %s", accessMethod)
	}
}

func printSessionInstructions(cmd *cobra.Command, client *aponoapi.AponoClient, session *clientapi.AccessSessionClientModel) error {
	accessDetails, customInstructionMessage, err := services.GetSessionDetails(cmd.Context(), client, session.Id, services.InstructionsOutputFormat)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(cmd.OutOrStdout(), "\n"+accessDetails)
	if err != nil {
		return err
	}

	if customInstructionMessage != "" {
		err = services.PrintCustomInstructionMessage(cmd, customInstructionMessage)
		if err != nil {
			return err
		}
	}

	if !services.IsSessionHaveNewCredentials(session) {
		err = printResetCredentialsSuggestion(cmd, session.Id)
		if err != nil {
			return err
		}
	}

	return nil
}

func printResetCredentialsSuggestion(cmd *cobra.Command, sessionID string) error {
	resetCommand := resetCredentialsCommand + sessionID
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "\n%s To get new set of credentials, run: %s\n", styles.NoticeMsgPrefix, color.Green.Sprint(resetCommand))
	return err
}

func PrintErrorConnectingSuggestion(cmd *cobra.Command, sessionID string) error {
	resetCommand := resetCredentialsCommand + sessionID
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "\n%s Problem to connect? Reset credentials using this command: %s\n\n", styles.NoticeMsgPrefix, color.Green.Sprint(resetCommand))
	return err
}
