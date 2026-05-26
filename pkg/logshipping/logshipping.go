package logshipping

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
)

const (
	LevelTrace = "TRACE"
	LevelDebug = "DEBUG"
	LevelInfo  = "INFO"
	LevelWarn  = "WARN"
	LevelError = "ERROR"

	callerCLI     = "cli"
	submitTimeout = 2 * time.Second
)

// sessionID groups all log entries emitted by a single CLI invocation. Stamped
// once at process start so support can find all events from one run.
var sessionID = uuid.NewString()

// Report sends one structured log event to the Apono backend.
//
// No-op when the context lacks an authenticated client (pre-login state).
// Failures of the underlying API call are silently dropped — telemetry must
// never affect the user-facing flow.
func Report(ctx context.Context, level, message string, fields map[string]string) {
	client, _ := aponoapi.GetClient(ctx)
	if client == nil {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, submitTimeout)
	defer cancel()

	entry := clientapi.LogEntryClientModel{
		SessionId: sessionID,
		Level:     level,
		Message:   message,
		Caller:    getCaller(),
		Timestamp: getTimestamp(),
		Fields:    fields,
	}
	_, _, _ = client.ClientAPI.LogsAPI.SubmitLogEntry(ctx).LogEntryClientModel(entry).Execute()
}

func getCaller() clientapi.NullableString {
	c := callerCLI
	return *clientapi.NewNullableString(&c)
}

func getTimestamp() clientapi.NullableFloat64 {
	ts := float64(time.Now().UnixNano()) / 1e9
	return *clientapi.NewNullableFloat64(&ts)
}
