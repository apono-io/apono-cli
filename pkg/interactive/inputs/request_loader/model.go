package requestloader

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/spinner"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
)

type model struct {
	spinner         spinner.Model
	ctx             context.Context
	client          *aponoapi.AponoClient
	request         *clientapi.AccessRequestClientModel
	creationTime    time.Time
	timeout         time.Duration
	lastRequestTime time.Time
	noWaitForGrant  bool
	quitting        bool
	aborting        bool
	err             error
}

type updatedRequestMsg clientapi.AccessRequestClientModel

type errMsg struct{ err error }
