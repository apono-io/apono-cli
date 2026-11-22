package actions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/build"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/connect"
	"github.com/apono-io/apono-cli/pkg/logshipping"
	"github.com/apono-io/apono-cli/pkg/services"
)

func TestResolveSessionIDArg(t *testing.T) {
	tests := []struct {
		name       string
		arg        string
		urlEncoded bool
		want       string
		wantErr    bool
	}{
		{name: "plain id from terminal is untouched", arg: "aws-rds-mysql-abc123", want: "aws-rds-mysql-abc123"},
		{name: "terminal arg with percent is not decoded", arg: "a%2Fb", want: "a%2Fb"},
		{name: "plain id from launch URL", arg: "aws-rds-mysql-abc123", urlEncoded: true, want: "aws-rds-mysql-abc123"},
		{
			name:       "encoded slashes from launch URL are decoded once",
			arg:        "aws-rds-mysql-dev%2Fmysql%2Feu1-payments-dev-commondev-aurora-cluster-d4bfac",
			urlEncoded: true,
			want:       "aws-rds-mysql-dev/mysql/eu1-payments-dev-commondev-aurora-cluster-d4bfac",
		},
		{name: "lowercase hex is accepted", arg: "a%2fb", urlEncoded: true, want: "a/b"},
		{name: "double-encoded is decoded only once", arg: "a%252Fb", urlEncoded: true, want: "a%2Fb"},
		{name: "plus sign is preserved", arg: "a+b", urlEncoded: true, want: "a+b"},
		{name: "malformed escape", arg: "a%zzb", urlEncoded: true, wantErr: true},
		{name: "truncated escape", arg: "a%2", urlEncoded: true, wantErr: true},
		{name: "control character after decode", arg: "a%0Ab", urlEncoded: true, wantErr: true},
		{name: "NUL after decode", arg: "a%00b", urlEncoded: true, wantErr: true},
		{name: "DEL after decode", arg: "a%7Fb", urlEncoded: true, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveSessionIDArg(tt.arg, tt.urlEncoded)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

const (
	useTestSessionID    = "session-1"
	useTestSessionName  = "prod-db"
	useTestSessionPath  = "/api/client/v1/access-sessions/" + useTestSessionID
	useTestDetailsPath  = useTestSessionPath + "/access-details"
	useTestLogsPath     = "/api/client/v1/logs"
	useTestConnected    = "connected"
	useTestNotFoundBody = `{"status":404,"message":"Session id or friendlyId session-1 not found"}`
	useTestServerError  = `{"status":500,"message":"boom"}`

	notActiveMessage     = `You don't have active access to prod-db. Request it again from the Apono portal or with "apono request create".`
	notFoundMessage      = "Access session session-1 was not found."
	notAvailableMessage  = "Access details for session session-1 are not available right now. Try again, or request access again if you no longer have it."
	detailsWrapPrefix    = "error getting access details for session id"
	resetSuggestionText  = "Reset credentials using this command"
	sessionNotActiveLog  = "access use: session not active"
	sessionNotFoundLog   = "access use: session not found"
	fetchSessionLog      = "access use: fetch session failed"
	fetchDetailsLog      = "access use: fetch session details failed"
	unsupportedErrorCode = "cli_unsupported_account"
	fieldCLIVersion      = "cli_version"
	harmlessCliCommand   = "echo launched-by-test"
)

type useStubResponse struct {
	status         int
	sessionStatus  string
	body           string
	headers        map[string]string
	dropConnection bool
}

type recordedLog struct {
	Level   string            `json:"level"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields"`
}

type useAPIStub struct {
	client          *aponoapi.AponoClient
	sessionRequests atomic.Int32
	detailsRequests atomic.Int32

	logsMutex sync.Mutex
	logs      []recordedLog
}

func (s *useAPIStub) failureLogs() []recordedLog {
	s.logsMutex.Lock()
	defer s.logsMutex.Unlock()

	var failures []recordedLog
	for _, entry := range s.logs {
		if entry.Level == logshipping.LevelWarn || entry.Level == logshipping.LevelError {
			failures = append(failures, entry)
		}
	}
	return failures
}

func (s *useAPIStub) recordLog(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	var entry recordedLog
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		t.Errorf("decode log entry: %v", err)
	}
	s.logsMutex.Lock()
	s.logs = append(s.logs, entry)
	s.logsMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{}`))
}

func useSessionBody(t *testing.T, status string) string {
	t.Helper()
	body, err := json.Marshal(clientapi.AccessSessionClientModel{
		Id:                useTestSessionID,
		Name:              useTestSessionName,
		ConnectionMethods: []string{services.CliOutputFormat},
		Status:            status,
		Launchers:         []clientapi.LauncherSummaryClientModel{},
		Credentials: *clientapi.NewNullableAccessSessionClientModelCredentials(&clientapi.AccessSessionClientModelCredentials{
			Id:       "credentials-1",
			Status:   "viewed",
			CanReset: true,
		}),
	})
	if err != nil {
		t.Fatalf("marshal session: %v", err)
	}
	return string(body)
}

func writeUseStubResponse(t *testing.T, w http.ResponseWriter, response useStubResponse) {
	t.Helper()
	if response.dropConnection {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			t.Error("response writer cannot be hijacked")
			return
		}
		conn, _, err := hijacker.Hijack()
		if err != nil {
			t.Errorf("hijack: %v", err)
			return
		}
		_ = conn.Close()
		return
	}

	body := response.body
	if response.sessionStatus != "" {
		body = useSessionBody(t, response.sessionStatus)
	}
	w.Header().Set("Content-Type", "application/json")
	for name, value := range response.headers {
		w.Header().Set(name, value)
	}
	w.WriteHeader(response.status)
	_, _ = w.Write([]byte(body))
}

func newUseAPIStub(t *testing.T, sessionResponses []useStubResponse, detailsResponse useStubResponse) *useAPIStub {
	t.Helper()
	stub := &useAPIStub{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case useTestSessionPath:
			requestIndex := int(stub.sessionRequests.Add(1)) - 1
			if requestIndex >= len(sessionResponses) {
				t.Errorf("unexpected session request number %d", requestIndex+1)
				w.WriteHeader(http.StatusTeapot)
				return
			}
			writeUseStubResponse(t, w, sessionResponses[requestIndex])
		case useTestDetailsPath:
			stub.detailsRequests.Add(1)
			writeUseStubResponse(t, w, detailsResponse)
		case useTestLogsPath:
			stub.recordLog(t, w, r)
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			w.WriteHeader(http.StatusTeapot)
		}
	}))
	t.Cleanup(server.Close)

	cfg := clientapi.NewConfiguration()
	cfg.Servers = clientapi.ServerConfigurations{{URL: server.URL}}
	cfg.HTTPClient = aponoapi.WithUnsupportedAccountCheck(&http.Client{Transport: &http.Transport{DisableKeepAlives: true}})
	stub.client = &aponoapi.AponoClient{ClientAPI: clientapi.NewAPIClient(cfg)}
	return stub
}

func runAccessUse(t *testing.T, stub *useAPIStub, flags ...string) (output string, err error) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv(accountIDEnvVar, "")

	var out bytes.Buffer
	cmd := AccessDetails()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetArgs(append(flags, "--", useTestSessionID))

	err = cmd.ExecuteContext(aponoapi.CreateClientContext(context.Background(), stub.client))
	return out.String(), err
}

type expectedFailureLog struct {
	level      string
	message    string
	clientID   string
	httpStatus string
	reason     string
}

func assertSingleFailureLog(t *testing.T, stub *useAPIStub, want expectedFailureLog) {
	t.Helper()
	failures := stub.failureLogs()
	if len(failures) != 1 {
		t.Fatalf("expected exactly 1 failure log, got %+v", failures)
	}
	failure := failures[0]
	if failure.Level != want.level || failure.Message != want.message {
		t.Fatalf("expected %s %q, got %s %q", want.level, want.message, failure.Level, failure.Message)
	}
	wantFields := map[string]string{
		fieldAccessSessionID:            useTestSessionID,
		fieldClientID:                   want.clientID,
		aponoapi.FailureFieldHTTPStatus: want.httpStatus,
		aponoapi.FailureFieldReason:     want.reason,
		fieldCLIVersion:                 build.Version,
	}
	if !maps.Equal(failure.Fields, wantFields) {
		t.Fatalf("expected fields %+v, got %+v", wantFields, failure.Fields)
	}
}

func assertErrorText(t *testing.T, err error, wantContains, wantExcludes []string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, text := range wantContains {
		if !strings.Contains(err.Error(), text) {
			t.Fatalf("expected error to contain %q, got %q", text, err.Error())
		}
	}
	for _, text := range wantExcludes {
		if strings.Contains(strings.ToLower(err.Error()), strings.ToLower(text)) {
			t.Fatalf("expected error not to contain %q, got %q", text, err.Error())
		}
	}
}

func TestAccessUse_sessionFetchFails_skipsAccessDetailsAndShipsOneLog(t *testing.T) {
	disconnected := useStubResponse{status: http.StatusOK, sessionStatus: connect.SessionStatusDisconnected}
	notFound := useStubResponse{status: http.StatusNotFound, body: useTestNotFoundBody}
	serverError := useStubResponse{status: http.StatusInternalServerError, body: useTestServerError}
	notActiveLog := expectedFailureLog{level: logshipping.LevelWarn, message: sessionNotActiveLog, reason: aponoapi.AccessSessionReasonNotActive}
	notFoundLog := expectedFailureLog{level: logshipping.LevelWarn, message: sessionNotFoundLog, httpStatus: "404", reason: aponoapi.AccessSessionReasonNotFound}
	serverErrorLog := expectedFailureLog{level: logshipping.LevelError, message: fetchSessionLog, httpStatus: "500"}
	transportLog := expectedFailureLog{level: logshipping.LevelError, message: fetchSessionLog}

	tests := []struct {
		name            string
		flags           []string
		sessionResponse useStubResponse
		wantErrorIs     string
		wantContains    []string
		wantExcludes    []string
		wantLog         expectedFailureLog
	}{
		{name: "disconnected with --run", flags: []string{"--run"}, sessionResponse: disconnected, wantErrorIs: notActiveMessage, wantLog: notActiveLog},
		{name: "disconnected with -o", flags: []string{"-o", "cli"}, sessionResponse: disconnected, wantErrorIs: notActiveMessage, wantLog: notActiveLog},
		{name: "not found with --run", flags: []string{"--run"}, sessionResponse: notFound, wantErrorIs: notFoundMessage, wantLog: notFoundLog},
		{name: "not found with -o", flags: []string{"-o", "cli"}, sessionResponse: notFound, wantErrorIs: notFoundMessage, wantLog: notFoundLog},
		{
			name:            "server error with --run",
			flags:           []string{"--run"},
			sessionResponse: serverError,
			wantContains:    []string{"500", "boom"},
			wantExcludes:    []string{"not found"},
			wantLog:         serverErrorLog,
		},
		{
			name:            "server error with -o",
			flags:           []string{"-o", "cli"},
			sessionResponse: serverError,
			wantContains:    []string{"500", "boom"},
			wantExcludes:    []string{"not found"},
			wantLog:         serverErrorLog,
		},
		{
			name:            "transport failure with -o",
			flags:           []string{"-o", "cli"},
			sessionResponse: useStubResponse{dropConnection: true},
			wantExcludes:    []string{"not found"},
			wantLog:         transportLog,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := newUseAPIStub(t, []useStubResponse{tt.sessionResponse}, useStubResponse{status: http.StatusOK})

			output, err := runAccessUse(t, stub, tt.flags...)

			assertErrorText(t, err, tt.wantContains, tt.wantExcludes)
			if tt.wantErrorIs != "" && err.Error() != tt.wantErrorIs {
				t.Fatalf("expected error %q, got %q", tt.wantErrorIs, err.Error())
			}
			if got := stub.detailsRequests.Load(); got != 0 {
				t.Fatalf("expected no access-details request, got %d", got)
			}
			if got := stub.sessionRequests.Load(); got != 1 {
				t.Fatalf("expected 1 session request, got %d", got)
			}
			if strings.Contains(output, resetSuggestionText) {
				t.Fatalf("expected no connect suggestion, got %q", output)
			}
			assertSingleFailureLog(t, stub, tt.wantLog)
		})
	}
}

func TestAccessUse_accessDetailsFetchFails_shipsOneLog(t *testing.T) {
	connected := useStubResponse{status: http.StatusOK, sessionStatus: useTestConnected}
	disconnected := useStubResponse{status: http.StatusOK, sessionStatus: connect.SessionStatusDisconnected}
	sessionNotFound := useStubResponse{status: http.StatusNotFound, body: useTestNotFoundBody}
	detailsNotFound := useStubResponse{status: http.StatusNotFound, body: useTestNotFoundBody}
	detailsServerError := useStubResponse{status: http.StatusInternalServerError, body: useTestServerError}
	notActiveLog := expectedFailureLog{level: logshipping.LevelWarn, message: fetchDetailsLog, httpStatus: "404", reason: aponoapi.AccessSessionReasonNotActive}
	serverErrorLog := expectedFailureLog{level: logshipping.LevelError, message: fetchDetailsLog, httpStatus: "500"}

	tests := []struct {
		name             string
		flags            []string
		sessionResponses []useStubResponse
		detailsResponse  useStubResponse
		wantErrorIs      string
		wantContains     []string
		wantExcludes     []string
		wantSuggestion   bool
		wantLog          expectedFailureLog
	}{
		{
			name:             "-o with 404 then disconnected session",
			flags:            []string{"-o", "cli"},
			sessionResponses: []useStubResponse{connected, disconnected},
			detailsResponse:  detailsNotFound,
			wantErrorIs:      notActiveMessage,
			wantLog:          notActiveLog,
		},
		{
			name:             "-o with a transport failure",
			flags:            []string{"-o", "cli"},
			sessionResponses: []useStubResponse{connected},
			detailsResponse:  useStubResponse{dropConnection: true},
			wantExcludes:     []string{"not found", "active access"},
			wantLog:          expectedFailureLog{level: logshipping.LevelError, message: fetchDetailsLog},
		},
		{
			name:             "-o with a server error",
			flags:            []string{"-o", "cli"},
			sessionResponses: []useStubResponse{connected},
			detailsResponse:  detailsServerError,
			wantErrorIs:      "500 Internal Server Error: boom",
			wantLog:          serverErrorLog,
		},
		{
			name:             "--run with 404 then disconnected session",
			flags:            []string{"--run"},
			sessionResponses: []useStubResponse{connected, disconnected},
			detailsResponse:  detailsNotFound,
			wantErrorIs:      notActiveMessage,
			wantExcludes:     []string{detailsWrapPrefix},
			wantLog:          notActiveLog,
		},
		{
			name:             "-o with 404 while the session is still connected",
			flags:            []string{"-o", "cli"},
			sessionResponses: []useStubResponse{connected, connected},
			detailsResponse:  detailsNotFound,
			wantErrorIs:      notAvailableMessage,
			wantLog:          expectedFailureLog{level: logshipping.LevelWarn, message: fetchDetailsLog, httpStatus: "404", reason: aponoapi.AccessSessionReasonUnknown},
		},
		{
			name:             "--run with 404 then session not found",
			flags:            []string{"--run"},
			sessionResponses: []useStubResponse{connected, sessionNotFound},
			detailsResponse:  detailsNotFound,
			wantErrorIs:      notFoundMessage,
			wantExcludes:     []string{detailsWrapPrefix},
			wantLog:          expectedFailureLog{level: logshipping.LevelWarn, message: fetchDetailsLog, httpStatus: "404", reason: aponoapi.AccessSessionReasonNotFound},
		},
		{
			name:             "--run with a server error keeps the wrap and the suggestion",
			flags:            []string{"--run"},
			sessionResponses: []useStubResponse{connected},
			detailsResponse:  detailsServerError,
			wantContains:     []string{detailsWrapPrefix + " " + useTestSessionID, "500 Internal Server Error: boom"},
			wantExcludes:     []string{"not found"},
			wantSuggestion:   true,
			wantLog:          serverErrorLog,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := newUseAPIStub(t, tt.sessionResponses, tt.detailsResponse)

			output, err := runAccessUse(t, stub, tt.flags...)

			assertErrorText(t, err, tt.wantContains, tt.wantExcludes)
			if tt.wantErrorIs != "" && err.Error() != tt.wantErrorIs {
				t.Fatalf("expected error %q, got %q", tt.wantErrorIs, err.Error())
			}
			if got := stub.detailsRequests.Load(); got != 1 {
				t.Fatalf("expected 1 access-details request, got %d", got)
			}
			if got := int(stub.sessionRequests.Load()); got != len(tt.sessionResponses) {
				t.Fatalf("expected %d session requests, got %d", len(tt.sessionResponses), got)
			}
			if printed := strings.Contains(output, resetSuggestionText); printed != tt.wantSuggestion {
				t.Fatalf("expected connect suggestion printed=%t, got output %q", tt.wantSuggestion, output)
			}
			assertSingleFailureLog(t, stub, tt.wantLog)
		})
	}
}

func TestAccessUse_unsupportedAccount_isReturnedAsItselfAndNotLogged(t *testing.T) {
	unsupportedMessage := "The Apono CLI doesn't support this account."
	stub := newUseAPIStub(t,
		[]useStubResponse{{
			status:  http.StatusForbidden,
			headers: map[string]string{"X-Apono-Error-Code": unsupportedErrorCode},
			body:    `{"status":403,"message":"` + unsupportedMessage + `"}`,
		}},
		useStubResponse{status: http.StatusOK},
	)

	_, err := runAccessUse(t, stub, "-o", "cli")

	var unsupportedErr *aponoapi.UnsupportedAccountError
	if !errors.As(err, &unsupportedErr) {
		t.Fatalf("expected an UnsupportedAccountError, got %T: %v", err, err)
	}
	if unsupportedErr.Message != unsupportedMessage {
		t.Fatalf("expected %q, got %q", unsupportedMessage, unsupportedErr.Message)
	}
	var apiErr *aponoapi.APIError
	if errors.As(err, &apiErr) {
		t.Fatalf("expected no APIError, got %v", apiErr)
	}
	if got := stub.detailsRequests.Load(); got != 0 {
		t.Fatalf("expected no access-details request, got %d", got)
	}
	if failures := stub.failureLogs(); len(failures) != 0 {
		t.Fatalf("expected no failure log, got %+v", failures)
	}
}

func useDetailsBody(t *testing.T) string {
	t.Helper()
	cliCommand := harmlessCliCommand
	body, err := json.Marshal(clientapi.AccessSessionDetailsClientModel{
		Instructions: clientapi.InstructionClientModel{Plain: "plain instructions", Markdown: "markdown instructions"},
		Cli:          *clientapi.NewNullableString(&cliCommand),
		Launchers:    []clientapi.LauncherClientModel{},
	})
	if err != nil {
		t.Fatalf("marshal access details: %v", err)
	}
	return string(body)
}

func TestAccessUse_connectedSession_printsOutputAndShipsNoFailureLog(t *testing.T) {
	tests := []struct {
		name       string
		flags      []string
		wantOutput string
	}{
		{name: "-o prints the cli command", flags: []string{"-o", "cli"}, wantOutput: harmlessCliCommand + "\n"},
		{name: "--run executes the cli command", flags: []string{"--run"}, wantOutput: "launched-by-test\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := newUseAPIStub(t,
				[]useStubResponse{{status: http.StatusOK, sessionStatus: useTestConnected}},
				useStubResponse{status: http.StatusOK, body: useDetailsBody(t)},
			)

			output, err := runAccessUse(t, stub, tt.flags...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if output != tt.wantOutput {
				t.Fatalf("expected output %q, got %q", tt.wantOutput, output)
			}
			if got := stub.sessionRequests.Load(); got != 1 {
				t.Fatalf("expected 1 session request, got %d", got)
			}
			if got := stub.detailsRequests.Load(); got != 1 {
				t.Fatalf("expected 1 access-details request, got %d", got)
			}
			if failures := stub.failureLogs(); len(failures) != 0 {
				t.Fatalf("expected no failure log, got %+v", failures)
			}
		})
	}
}

func TestAccessUse_clientFlag_sessionFetchFails_shipsOneLauncherLog(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("--client is only supported on macOS")
	}

	tests := []struct {
		name            string
		sessionResponse useStubResponse
		wantErrorIs     string
		wantContains    []string
		wantExcludes    []string
		wantLog         expectedFailureLog
	}{
		{
			name:            "disconnected session",
			sessionResponse: useStubResponse{status: http.StatusOK, sessionStatus: connect.SessionStatusDisconnected},
			wantErrorIs:     notActiveMessage,
			wantLog:         expectedFailureLog{level: logshipping.LevelWarn, message: "launcher: session not active", reason: aponoapi.AccessSessionReasonNotActive},
		},
		{
			name:            "session not found",
			sessionResponse: useStubResponse{status: http.StatusNotFound, body: useTestNotFoundBody},
			wantErrorIs:     notFoundMessage,
			wantLog:         expectedFailureLog{level: logshipping.LevelWarn, message: "launcher: session not found", httpStatus: "404", reason: aponoapi.AccessSessionReasonNotFound},
		},
		{
			name:            "server error",
			sessionResponse: useStubResponse{status: http.StatusInternalServerError, body: useTestServerError},
			wantErrorIs:     "500 Internal Server Error: boom",
			wantExcludes:    []string{"not found"},
			wantLog:         expectedFailureLog{level: logshipping.LevelError, message: "launcher: fetch session failed", httpStatus: "500"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := newUseAPIStub(t, []useStubResponse{tt.sessionResponse}, useStubResponse{status: http.StatusOK})

			_, err := runAccessUse(t, stub, "--client", "dbeaver")

			assertErrorText(t, err, tt.wantContains, tt.wantExcludes)
			if err.Error() != tt.wantErrorIs {
				t.Fatalf("expected error %q, got %q", tt.wantErrorIs, err.Error())
			}
			if got := stub.detailsRequests.Load(); got != 0 {
				t.Fatalf("expected no access-details request, got %d", got)
			}
			wantLog := tt.wantLog
			wantLog.clientID = "dbeaver"
			assertSingleFailureLog(t, stub, wantLog)
		})
	}
}
