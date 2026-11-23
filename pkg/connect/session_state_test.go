package connect

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/clientapi"
)

const (
	stateTestSessionID   = "session-1"
	stateTestSessionName = "prod-db"
	stateTestSessionPath = "/api/client/v1/access-sessions/" + stateTestSessionID
	stateTestDetailsPath = stateTestSessionPath + "/access-details"
	statusConnected      = "connected"
)

type stubResponse struct {
	status         int
	sessionStatus  string
	body           string
	headers        map[string]string
	dropConnection bool
	beforeResponse func(r *http.Request)
}

type sessionAPIStub struct {
	client          *aponoapi.AponoClient
	sessionRequests atomic.Int32
	detailsRequests atomic.Int32
}

func sessionBody(t *testing.T, status string) string {
	t.Helper()
	body, err := json.Marshal(clientapi.AccessSessionClientModel{
		Id:                stateTestSessionID,
		Name:              stateTestSessionName,
		ConnectionMethods: []string{},
		Status:            status,
		Launchers:         []clientapi.LauncherSummaryClientModel{},
	})
	if err != nil {
		t.Fatalf("marshal session: %v", err)
	}
	return string(body)
}

func writeStubResponse(t *testing.T, w http.ResponseWriter, r *http.Request, response stubResponse) {
	t.Helper()
	if response.beforeResponse != nil {
		response.beforeResponse(r)
	}
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
		body = sessionBody(t, response.sessionStatus)
	}
	w.Header().Set("Content-Type", "application/json")
	for name, value := range response.headers {
		w.Header().Set(name, value)
	}
	w.WriteHeader(response.status)
	_, _ = w.Write([]byte(body))
}

func newSessionAPIStub(t *testing.T, sessionResponse, detailsResponse stubResponse) *sessionAPIStub {
	t.Helper()
	stub := &sessionAPIStub{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case stateTestSessionPath:
			stub.sessionRequests.Add(1)
			writeStubResponse(t, w, r, sessionResponse)
		case stateTestDetailsPath:
			stub.detailsRequests.Add(1)
			writeStubResponse(t, w, r, detailsResponse)
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

type expectedFailure struct {
	reason      string
	httpStatus  int
	sessionName string
	isAPIError  bool
	message     string
}

func assertFailure(t *testing.T, err error, want expectedFailure) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error")
	}

	var apiErr *aponoapi.APIError
	var unavailableErr *aponoapi.AccessSessionUnavailableError

	if want.isAPIError {
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected an APIError, got %T: %v", err, err)
		}
		if apiErr.HTTPStatus != want.httpStatus {
			t.Fatalf("expected status %d, got %d", want.httpStatus, apiErr.HTTPStatus)
		}
		if want.message != "" && apiErr.Error() != want.message {
			t.Fatalf("expected %q, got %q", want.message, apiErr.Error())
		}
		return
	}

	if !errors.As(err, &unavailableErr) {
		t.Fatalf("expected an AccessSessionUnavailableError, got %T: %v", err, err)
	}
	if unavailableErr.Reason != want.reason {
		t.Fatalf("expected reason %q, got %q", want.reason, unavailableErr.Reason)
	}
	if unavailableErr.HTTPStatus != want.httpStatus {
		t.Fatalf("expected status %d, got %d", want.httpStatus, unavailableErr.HTTPStatus)
	}
	if unavailableErr.SessionID != stateTestSessionID {
		t.Fatalf("expected session id %q, got %q", stateTestSessionID, unavailableErr.SessionID)
	}
	if unavailableErr.SessionName != want.sessionName {
		t.Fatalf("expected session name %q, got %q", want.sessionName, unavailableErr.SessionName)
	}
}

func TestFetchSession(t *testing.T) {
	tests := []struct {
		name            string
		sessionResponse stubResponse
		wantSession     bool
		wantFailure     expectedFailure
	}{
		{
			name:            "connected session is returned",
			sessionResponse: stubResponse{status: http.StatusOK, sessionStatus: statusConnected},
			wantSession:     true,
		},
		{
			name:            "disconnected session is not active",
			sessionResponse: stubResponse{status: http.StatusOK, sessionStatus: SessionStatusDisconnected},
			wantFailure:     expectedFailure{reason: aponoapi.AccessSessionReasonNotActive, sessionName: stateTestSessionName},
		},
		{
			name:            "disconnected status is matched case insensitively",
			sessionResponse: stubResponse{status: http.StatusOK, sessionStatus: strings.ToUpper(SessionStatusDisconnected)},
			wantFailure:     expectedFailure{reason: aponoapi.AccessSessionReasonNotActive, sessionName: stateTestSessionName},
		},
		{
			name:            "404 is not found",
			sessionResponse: stubResponse{status: http.StatusNotFound, body: `{"status":404,"message":"Session id or friendlyId session-1 not found"}`},
			wantFailure:     expectedFailure{reason: aponoapi.AccessSessionReasonNotFound, httpStatus: http.StatusNotFound},
		},
		{
			name:            "500 is an api error",
			sessionResponse: stubResponse{status: http.StatusInternalServerError, body: `{"status":500,"message":"boom"}`},
			wantFailure:     expectedFailure{isAPIError: true, httpStatus: http.StatusInternalServerError, message: "500 Internal Server Error: boom"},
		},
		{
			name:            "transport failure is an api error without a status",
			sessionResponse: stubResponse{dropConnection: true},
			wantFailure:     expectedFailure{isAPIError: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := newSessionAPIStub(t, tt.sessionResponse, stubResponse{status: http.StatusOK})

			session, err := FetchSession(context.Background(), stub.client, stateTestSessionID)

			if got := stub.sessionRequests.Load(); got != 1 {
				t.Fatalf("expected 1 session request, got %d", got)
			}
			if got := stub.detailsRequests.Load(); got != 0 {
				t.Fatalf("expected no access-details request, got %d", got)
			}

			if tt.wantSession {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if session == nil || session.Id != stateTestSessionID {
					t.Fatalf("expected session %q, got %+v", stateTestSessionID, session)
				}
				return
			}

			if session != nil {
				t.Fatalf("expected no session, got %+v", session)
			}
			assertFailure(t, err, tt.wantFailure)
		})
	}
}

func TestFetchAccessDetailsClassifiesFailures(t *testing.T) {
	notFoundBody := `{"status":404,"message":"Session id or friendlyId session-1 not found"}`

	tests := []struct {
		name                string
		detailsResponse     stubResponse
		sessionResponse     stubResponse
		wantSessionRequests int32
		wantFailure         expectedFailure
	}{
		{
			name:                "404 then disconnected session is not active",
			detailsResponse:     stubResponse{status: http.StatusNotFound, body: notFoundBody},
			sessionResponse:     stubResponse{status: http.StatusOK, sessionStatus: SessionStatusDisconnected},
			wantSessionRequests: 1,
			wantFailure:         expectedFailure{reason: aponoapi.AccessSessionReasonNotActive, httpStatus: http.StatusNotFound, sessionName: stateTestSessionName},
		},
		{
			name:                "404 then 404 is not found",
			detailsResponse:     stubResponse{status: http.StatusNotFound, body: notFoundBody},
			sessionResponse:     stubResponse{status: http.StatusNotFound, body: notFoundBody},
			wantSessionRequests: 1,
			wantFailure:         expectedFailure{reason: aponoapi.AccessSessionReasonNotFound, httpStatus: http.StatusNotFound},
		},
		{
			name:                "404 then connected session is unknown",
			detailsResponse:     stubResponse{status: http.StatusNotFound, body: notFoundBody},
			sessionResponse:     stubResponse{status: http.StatusOK, sessionStatus: statusConnected},
			wantSessionRequests: 1,
			wantFailure:         expectedFailure{reason: aponoapi.AccessSessionReasonUnknown, httpStatus: http.StatusNotFound},
		},
		{
			name:                "404 then failing re-fetch is unknown",
			detailsResponse:     stubResponse{status: http.StatusNotFound, body: notFoundBody},
			sessionResponse:     stubResponse{status: http.StatusInternalServerError, body: `{"status":500,"message":"boom"}`},
			wantSessionRequests: 1,
			wantFailure:         expectedFailure{reason: aponoapi.AccessSessionReasonUnknown, httpStatus: http.StatusNotFound},
		},
		{
			name:                "404 then re-fetch transport failure is unknown",
			detailsResponse:     stubResponse{status: http.StatusNotFound, body: notFoundBody},
			sessionResponse:     stubResponse{dropConnection: true},
			wantSessionRequests: 1,
			wantFailure:         expectedFailure{reason: aponoapi.AccessSessionReasonUnknown, httpStatus: http.StatusNotFound},
		},
		{
			name:            "500 is an api error without a re-fetch",
			detailsResponse: stubResponse{status: http.StatusInternalServerError, body: `{"status":500,"message":"boom"}`},
			sessionResponse: stubResponse{status: http.StatusOK, sessionStatus: SessionStatusDisconnected},
			wantFailure:     expectedFailure{isAPIError: true, httpStatus: http.StatusInternalServerError, message: "500 Internal Server Error: boom"},
		},
		{
			name:            "transport failure is an api error without a re-fetch",
			detailsResponse: stubResponse{dropConnection: true},
			sessionResponse: stubResponse{status: http.StatusOK, sessionStatus: SessionStatusDisconnected},
			wantFailure:     expectedFailure{isAPIError: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := newSessionAPIStub(t, tt.sessionResponse, tt.detailsResponse)

			details, err := FetchAccessDetails(context.Background(), stub.client, stateTestSessionID)

			if details != nil {
				t.Fatalf("expected no details, got %+v", details)
			}
			if got := stub.detailsRequests.Load(); got != 1 {
				t.Fatalf("expected 1 access-details request, got %d", got)
			}
			if got := stub.sessionRequests.Load(); got != tt.wantSessionRequests {
				t.Fatalf("expected %d session requests, got %d", tt.wantSessionRequests, got)
			}
			assertFailure(t, err, tt.wantFailure)
		})
	}
}

func TestClassifyAccessDetailsErrorKeepsUnsupportedAccountError(t *testing.T) {
	stub := newSessionAPIStub(t, stubResponse{status: http.StatusOK, sessionStatus: statusConnected}, stubResponse{status: http.StatusOK})
	unsupportedErr := &aponoapi.UnsupportedAccountError{Message: "The Apono CLI doesn't support this account."}

	classified := ClassifyAccessDetailsError(context.Background(), stub.client, stateTestSessionID, unsupportedErr, nil)

	if !errors.Is(classified, unsupportedErr) {
		t.Fatalf("expected the unsupported account error untouched, got %T: %v", classified, classified)
	}
	if got := stub.sessionRequests.Load(); got != 0 {
		t.Fatalf("expected no session request, got %d", got)
	}
}

func TestFetchAccessDetailsKeepsUnsupportedAccountErrorFromTheRefetch(t *testing.T) {
	unsupportedMessage := "The Apono CLI doesn't support this account."
	stub := newSessionAPIStub(t,
		stubResponse{
			status:  http.StatusForbidden,
			headers: map[string]string{"X-Apono-Error-Code": "cli_unsupported_account"},
			body:    `{"status":403,"message":"` + unsupportedMessage + `"}`,
		},
		stubResponse{status: http.StatusNotFound, body: `{"status":404,"message":"not found"}`},
	)

	_, err := FetchAccessDetails(context.Background(), stub.client, stateTestSessionID)

	unsupportedErr, isUnsupportedAccount := aponoapi.AsUnsupportedAccountError(err)
	if !isUnsupportedAccount {
		t.Fatalf("expected an unsupported account error, got %T: %v", err, err)
	}
	if unsupportedErr.Message != unsupportedMessage {
		t.Fatalf("expected %q, got %q", unsupportedMessage, unsupportedErr.Message)
	}
	var unavailableErr *aponoapi.AccessSessionUnavailableError
	if errors.As(err, &unavailableErr) {
		t.Fatalf("expected no AccessSessionUnavailableError, got reason %q", unavailableErr.Reason)
	}
	if got := stub.sessionRequests.Load(); got != 1 {
		t.Fatalf("expected 1 session request, got %d", got)
	}
}

type manuallyExpiredContext struct {
	context.Context
	expired chan struct{}
}

func (c *manuallyExpiredContext) Done() <-chan struct{} {
	return c.expired
}

func (c *manuallyExpiredContext) Err() error {
	select {
	case <-c.expired:
		return context.DeadlineExceeded
	default:
		return nil
	}
}

func TestFetchAccessDetailsKeepsContextErrorFromTheRefetch(t *testing.T) {
	tests := []struct {
		name       string
		newContext func() (ctx context.Context, end func())
		wantErr    error
	}{
		{
			name: "canceled",
			newContext: func() (context.Context, func()) {
				return context.WithCancel(context.Background())
			},
			wantErr: context.Canceled,
		},
		{
			name: "deadline exceeded",
			newContext: func() (context.Context, func()) {
				ctx := &manuallyExpiredContext{Context: context.Background(), expired: make(chan struct{})}
				return ctx, sync.OnceFunc(func() { close(ctx.expired) })
			},
			wantErr: context.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, endContext := tt.newContext()
			defer endContext()

			stub := newSessionAPIStub(t,
				stubResponse{
					dropConnection: true,
					beforeResponse: func(r *http.Request) {
						endContext()
						<-r.Context().Done()
					},
				},
				stubResponse{status: http.StatusNotFound, body: `{"status":404,"message":"not found"}`},
			)

			_, err := FetchAccessDetails(ctx, stub.client, stateTestSessionID)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %T: %v", tt.wantErr, err, err)
			}
			var unavailableErr *aponoapi.AccessSessionUnavailableError
			if errors.As(err, &unavailableErr) {
				t.Fatalf("expected no AccessSessionUnavailableError, got reason %q", unavailableErr.Reason)
			}
			if got := stub.sessionRequests.Load(); got != 1 {
				t.Fatalf("expected 1 session request, got %d", got)
			}
		})
	}
}
