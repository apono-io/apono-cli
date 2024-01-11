package requestloader

import (
	"context"
	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/charmbracelet/bubbles/spinner"
	"time"
)

type model struct {
	spinner        spinner.Model
	ctx            context.Context
	client         *aponoapi.AponoClient
	request        *clientapi.AccessRequestClientModel
	creationTime   time.Time
	timeout        time.Duration
	noWaitForGrant bool
	quitting       bool
	aborting       bool
	err            error
}

type statusMsg clientapi.AccessRequestClientModel

type errMsg struct{ err error }
