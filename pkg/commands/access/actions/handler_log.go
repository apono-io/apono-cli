package actions

import (
	"context"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/logshipping"
	"github.com/apono-io/apono-cli/pkg/urihandler"
)

// shipHandlerLog drains the apono:// handler's trace file and forwards each line
// to the backend under the handler caller. Best-effort: it does nothing without
// an authenticated client (draining would empty the file with nowhere to ship),
// and swallows any file error.
func shipHandlerLog(ctx context.Context) {
	if client, _ := aponoapi.GetClient(ctx); client == nil {
		return
	}
	path, err := urihandler.HandlerLogPath()
	if err != nil {
		return
	}
	lines, err := urihandler.DrainLog(path)
	if err != nil {
		return
	}
	for _, line := range lines {
		logshipping.Report(ctx, logshipping.CallerHandler, line.Level, line.Message, nil)
	}
}
