package aponoapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/oauth2"

	"github.com/apono-io/apono-cli/pkg/clientapi"
)

func TestAccessSessionUnavailableErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		err  AccessSessionUnavailableError
		want string
	}{
		{
			name: "not active uses the session name",
			err:  AccessSessionUnavailableError{Reason: AccessSessionReasonNotActive, SessionID: "session-1", SessionName: "prod-db"},
			want: `You don't have active access to prod-db. Request it again from the Apono portal or with "apono request create".`,
		},
		{
			name: "not active falls back to the session id",
			err:  AccessSessionUnavailableError{Reason: AccessSessionReasonNotActive, SessionID: "session-1"},
			want: `You don't have active access to session-1. Request it again from the Apono portal or with "apono request create".`,
		},
		{
			name: "not found",
			err:  AccessSessionUnavailableError{Reason: AccessSessionReasonNotFound, SessionID: "session-1", SessionName: "prod-db", HTTPStatus: http.StatusNotFound},
			want: "Access session session-1 was not found.",
		},
		{
			name: "unknown",
			err:  AccessSessionUnavailableError{Reason: AccessSessionReasonUnknown, SessionID: "session-1", HTTPStatus: http.StatusNotFound},
			want: "Access details for session session-1 are not available right now. Try again, or request access again if you no longer have it.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestDescribeAPIErrorForHTTPFailures(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		body        string
		wantMessage string
	}{
		{name: "json body with message", status: http.StatusInternalServerError, body: `{"status":500,"message":"database is down"}`, wantMessage: "500 Internal Server Error: database is down"},
		{name: "json body without message", status: http.StatusBadGateway, body: `{"status":502}`, wantMessage: "502 Bad Gateway"},
		{name: "non json body", status: http.StatusServiceUnavailable, body: "<html>unavailable</html>", wantMessage: "503 Service Unavailable"},
		{name: "empty body", status: http.StatusForbidden, body: "", wantMessage: "403 Forbidden"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			cfg := clientapi.NewConfiguration()
			cfg.Servers = clientapi.ServerConfigurations{{URL: server.URL}}
			client := &AponoClient{ClientAPI: clientapi.NewAPIClient(cfg)}

			_, resp, callErr := client.ClientAPI.AccessSessionsAPI.GetAccessSession(context.Background(), "session-1").Execute()
			if callErr != nil && resp != nil {
				defer func() {
					_ = resp.Body.Close()
				}()
			}
			if callErr == nil {
				t.Fatal("expected the api call to fail")
			}

			var apiErr *APIError
			if !errors.As(DescribeAPIError(callErr, resp), &apiErr) {
				t.Fatalf("expected an APIError, got %T", DescribeAPIError(callErr, resp))
			}
			if apiErr.HTTPStatus != tt.status {
				t.Fatalf("expected status %d, got %d", tt.status, apiErr.HTTPStatus)
			}
			if apiErr.Error() != tt.wantMessage {
				t.Fatalf("expected %q, got %q", tt.wantMessage, apiErr.Error())
			}
		})
	}
}

func TestDescribeAPIErrorWithoutResponse(t *testing.T) {
	unsupportedErr := &UnsupportedAccountError{Message: unsupportedMessage}
	wrappedUnsupportedErr := fmt.Errorf("Get \"https://api.apono.io\": %w", unsupportedErr)
	transportErr := errors.New("dial tcp 127.0.0.1:443: connect: connection refused")

	tests := []struct {
		name            string
		err             error
		wantPassThrough bool
		wantMessage     string
	}{
		{name: "transport failure", err: transportErr, wantMessage: transportErr.Error()},
		{name: "unsupported account", err: unsupportedErr, wantPassThrough: true},
		{name: "wrapped unsupported account", err: wrappedUnsupportedErr, wantPassThrough: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			described := DescribeAPIError(tt.err, nil)

			var apiErr *APIError
			isAPIError := errors.As(described, &apiErr)

			if tt.wantPassThrough {
				if described != tt.err {
					t.Fatalf("expected the original error untouched, got %v", described)
				}
				if isAPIError {
					t.Fatal("expected no APIError for an unsupported account")
				}
				return
			}

			if !isAPIError {
				t.Fatalf("expected an APIError, got %T", described)
			}
			if apiErr.HTTPStatus != 0 {
				t.Fatalf("expected status 0, got %d", apiErr.HTTPStatus)
			}
			if apiErr.Error() != tt.wantMessage {
				t.Fatalf("expected %q, got %q", tt.wantMessage, apiErr.Error())
			}
		})
	}
}

func TestFailureFields(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		wantHTTPStatus string
		wantReason     string
	}{
		{
			name:           "unavailable with status",
			err:            &AccessSessionUnavailableError{Reason: AccessSessionReasonNotFound, HTTPStatus: http.StatusNotFound},
			wantHTTPStatus: "404",
			wantReason:     AccessSessionReasonNotFound,
		},
		{
			name:       "unavailable without a failed call",
			err:        &AccessSessionUnavailableError{Reason: AccessSessionReasonNotActive},
			wantReason: AccessSessionReasonNotActive,
		},
		{
			name:           "wrapped unavailable",
			err:            fmt.Errorf("could not fetch session details: %w", &AccessSessionUnavailableError{Reason: AccessSessionReasonUnknown, HTTPStatus: http.StatusNotFound}),
			wantHTTPStatus: "404",
			wantReason:     AccessSessionReasonUnknown,
		},
		{
			name:           "api error with status",
			err:            &APIError{HTTPStatus: http.StatusInternalServerError, Message: "500 Internal Server Error"},
			wantHTTPStatus: "500",
		},
		{
			name:           "wrapped api error",
			err:            fmt.Errorf("error getting access details: %w", &APIError{HTTPStatus: http.StatusBadGateway, Message: "502 Bad Gateway"}),
			wantHTTPStatus: "502",
		},
		{
			name: "api error for a transport failure",
			err:  &APIError{Message: "connection refused"},
		},
		{
			name: "plain error",
			err:  errors.New("command failed"),
		},
		{
			name: "nil error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpStatus, reason := FailureFields(tt.err)
			if httpStatus != tt.wantHTTPStatus {
				t.Fatalf("expected http status %q, got %q", tt.wantHTTPStatus, httpStatus)
			}
			if reason != tt.wantReason {
				t.Fatalf("expected reason %q, got %q", tt.wantReason, reason)
			}
		})
	}
}

func describeFailedSessionCall(t *testing.T, status int, body string) (described, callErr error) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	cfg := clientapi.NewConfiguration()
	cfg.Servers = clientapi.ServerConfigurations{{URL: server.URL}}
	client := &AponoClient{ClientAPI: clientapi.NewAPIClient(cfg)}

	_, resp, callErr := client.ClientAPI.AccessSessionsAPI.GetAccessSession(context.Background(), "session-1").Execute()
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if callErr == nil {
		t.Fatal("expected the api call to fail")
	}

	return DescribeAPIError(callErr, resp), callErr
}

func TestDescribeAPIErrorSanitizesTheServerMessage(t *testing.T) {
	tests := []struct {
		name          string
		serverMessage string
		wantMessage   string
	}{
		{
			name:          "control characters are stripped",
			serverMessage: "line one\nline two\ttabbed\u001b[31m red\u0000\u007f",
			wantMessage:   "500 Internal Server Error: line oneline twotabbed[31m red",
		},
		{
			name:          "long message is capped",
			serverMessage: strings.Repeat("é", serverMessageMaxLength+100),
			wantMessage:   "500 Internal Server Error: " + strings.Repeat("é", serverMessageMaxLength),
		},
		{
			name:          "message at the cap is kept whole",
			serverMessage: strings.Repeat("a", serverMessageMaxLength),
			wantMessage:   "500 Internal Server Error: " + strings.Repeat("a", serverMessageMaxLength),
		},
		{
			name:          "message of only control characters falls back to the status line",
			serverMessage: "\n\r\t",
			wantMessage:   "500 Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, marshalErr := json.Marshal(map[string]any{"status": http.StatusInternalServerError, "message": tt.serverMessage})
			if marshalErr != nil {
				t.Fatalf("marshal body: %v", marshalErr)
			}

			described, _ := describeFailedSessionCall(t, http.StatusInternalServerError, string(body))

			var apiErr *APIError
			if !errors.As(described, &apiErr) {
				t.Fatalf("expected an APIError, got %T", described)
			}
			if apiErr.Error() != tt.wantMessage {
				t.Fatalf("expected %q, got %q", tt.wantMessage, apiErr.Error())
			}
		})
	}
}

func TestDescribeAPIErrorForSuccessStatusWithUndecodableBody(t *testing.T) {
	described, callErr := describeFailedSessionCall(t, http.StatusOK, "not json")

	var apiErr *APIError
	if !errors.As(described, &apiErr) {
		t.Fatalf("expected an APIError, got %T", described)
	}
	if apiErr.HTTPStatus != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, apiErr.HTTPStatus)
	}
	if apiErr.Error() == "" || apiErr.Error() != callErr.Error() {
		t.Fatalf("expected the decode error text %q, got %q", callErr.Error(), apiErr.Error())
	}
}

func TestDescribeAPIErrorWithNilError(t *testing.T) {
	responses := map[string]*http.Response{
		"without a response": nil,
		"with a response":    {StatusCode: http.StatusInternalServerError, Status: "500 Internal Server Error"},
	}

	for name, resp := range responses {
		t.Run(name, func(t *testing.T) {
			if described := DescribeAPIError(nil, resp); described != nil {
				t.Fatalf("expected nil, got %T: %v", described, described)
			}
		})
	}
}

func TestDescribeAPIErrorKeepsTheOriginalErrorReachable(t *testing.T) {
	expiredLogin := &oauth2.RetrieveError{Body: []byte(`{"error":"invalid_grant"}`)}

	described := DescribeAPIError(fmt.Errorf("fetch session: %w", expiredLogin), nil)
	if !IsInvalidGrant(described) {
		t.Fatalf("expected the expired login to stay detectable, got %v", described)
	}

	canceled := DescribeAPIError(context.Canceled, nil)
	if !errors.Is(canceled, context.Canceled) {
		t.Fatalf("expected the cancellation to stay detectable, got %v", canceled)
	}
}
