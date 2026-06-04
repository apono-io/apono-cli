// Package logshipping forwards CLI-side error context to the Apono backend.
//
// Callers invoke Report at error sites. Each call fires a single synchronous
// API request with a short timeout. Network errors are swallowed — diagnostic
// telemetry must not affect the user-facing flow.
package logshipping

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/build"
	"github.com/apono-io/apono-cli/pkg/clientapi"
)

const (
	LevelTrace = "TRACE"
	LevelDebug = "DEBUG"
	LevelInfo  = "INFO"
	LevelWarn  = "WARN"
	LevelError = "ERROR"

	callerCLI       = "cli"
	fieldCLIVersion = "cli_version"
	submitTimeout   = 2 * time.Second

	// maxEventsPerInvocation caps how many events one CLI process ships, so a
	// runaway error loop can't flood the backend.
	maxEventsPerInvocation = 10
)

var (
	sessionID  = uuid.NewString()
	eventCount atomic.Int32
)

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
	if !withinCap(eventCount.Add(1)) {
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
		Fields:    withCLIVersion(fields),
	}
	_ = submit(ctx, client, entry)
}

// withinCap reports whether the nth event (1-based) is within the
// per-invocation ship cap.
func withinCap(count int32) bool {
	return count <= maxEventsPerInvocation
}

// withCLIVersion returns a copy of fields with the CLI build version added,
// without mutating the caller's map.
func withCLIVersion(fields map[string]string) map[string]string {
	enriched := make(map[string]string, len(fields)+1)
	for k, v := range fields {
		enriched[k] = v
	}
	enriched[fieldCLIVersion] = build.Version
	return enriched
}

func submit(ctx context.Context, client *aponoapi.AponoClient, entry clientapi.LogEntryClientModel) error {
	_, _, err := client.ClientAPI.LogsAPI.SubmitLogEntry(ctx).LogEntryClientModel(entry).Execute()
	return err
}

func getCaller() clientapi.NullableString {
	c := callerCLI
	return *clientapi.NewNullableString(&c)
}

func getTimestamp() clientapi.NullableFloat64 {
	ts := float64(time.Now().UnixNano()) / 1e9
	return *clientapi.NewNullableFloat64(&ts)
}
