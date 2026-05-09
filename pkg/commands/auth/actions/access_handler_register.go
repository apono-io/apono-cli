package actions

import (
	"fmt"
	"io"
	"runtime"

	"github.com/apono-io/apono-cli/pkg/config"
	"github.com/apono-io/apono-cli/pkg/terminal"
	"github.com/apono-io/apono-cli/pkg/urihandler"
)

const darwinOS = "darwin"

func setupAccessHandler(in io.Reader, out io.Writer) error {
	if runtime.GOOS != darwinOS {
		return nil
	}
	if !terminal.IsRunning(in) {
		return nil
	}
	if config.IsAccessHandlerAnnounced() {
		return nil
	}

	if _, err := fmt.Fprintln(out, "\nInstalling the apono:// URL handler. Required to open sessions launched from the Apono portal and Slack."); err != nil {
		return err
	}

	if err := config.MarkAccessHandlerAnnounced(); err != nil {
		return fmt.Errorf("persist access-handler state: %w", err)
	}

	if err := urihandler.Register(out); err != nil {
		return fmt.Errorf("%w (retry later with `apono access-handler register`)", err)
	}
	return nil
}
