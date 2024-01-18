package flows

import (
	"fmt"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/interactive"
	"github.com/apono-io/apono-cli/pkg/services"
	"github.com/apono-io/apono-cli/pkg/styles"

	"github.com/gookit/color"
	"github.com/spf13/cobra"
)

func RunSessionInteractiveFlow(cmd *cobra.Command, client *aponoapi.AponoClient, requestID string) error {
	session, err := interactive.RunSessionsSelector(cmd.Context(), client, requestID)
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

	accessMethod, err := interactive.RunSessionCliMethodOptionSelector()
	if err != nil {
		return err
	}

	switch accessMethod {
	case interactive.ExecuteOption:
		err = services.ExecuteAccessDetails(cmd, client, session)
		return err

	case interactive.PrintOption:
		err = printSessionInstructions(cmd, client, session)
		return err

	default:
		return fmt.Errorf("unknown access method %s", accessMethod)
	}
}

func printSessionInstructions(cmd *cobra.Command, client *aponoapi.AponoClient, session *clientapi.AccessSessionClientModel) error {
	accessDetails, err := services.GetSessionDetails(cmd.Context(), client, session.Id, services.InstructionsOutputFormat)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(cmd.OutOrStdout(), "\n"+accessDetails)
	if err != nil {
		return err
	}

	if !services.IsSessionHaveNewCredentials(session) {
		err = suggestResetCredentialsCommand(cmd, session.Id)
		if err != nil {
			return err
		}
	}

	return nil
}

func suggestResetCredentialsCommand(cmd *cobra.Command, sessionID string) error {
	resetCommand := "apono access reset-credentials " + sessionID
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "\n%s To get new set of credentials, run: %s\n", styles.NoticeMsgPrefix, color.Green.Sprint(resetCommand))
	return err
}
