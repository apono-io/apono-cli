package aponoapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/oauth2"
)

const unsupportedMessage = "The Apono CLI doesn't support 1Password Privileged Access accounts.\nGo to the web portal"

func TestUnsupportedAccountTransport(t *testing.T) {
	tests := []struct {
		name        string
		header      string
		body        string
		wantMessage string
	}{
		{name: "tagged response with message", header: unsupportedAccountErrorCode, body: `{"status":403,"message":"` + "The Apono CLI doesn't support 1Password Privileged Access accounts.\\nGo to the web portal" + `"}`, wantMessage: unsupportedMessage},
		{name: "tagged response without message", header: unsupportedAccountErrorCode, body: `not json`, wantMessage: unsupportedAccountFallback},
		{name: "untagged 403 passes through", header: "", body: `{"message":"forbidden"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tt.header != "" {
					w.Header().Set(errorCodeHeader, tt.header)
				}
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			req, reqErr := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
			if reqErr != nil {
				t.Fatal(reqErr)
			}
			resp, err := WithUnsupportedAccountCheck(&http.Client{}).Do(req)
			if tt.wantMessage == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				_ = resp.Body.Close()
				if resp.StatusCode != http.StatusForbidden {
					t.Fatalf("expected 403, got %d", resp.StatusCode)
				}
				return
			}

			var unsupportedErr *UnsupportedAccountError
			if !errors.As(err, &unsupportedErr) {
				t.Fatalf("expected UnsupportedAccountError, got %v", err)
			}
			if unsupportedErr.Message != tt.wantMessage {
				t.Fatalf("expected message %q, got %q", tt.wantMessage, unsupportedErr.Message)
			}
		})
	}
}

func TestAsUnsupportedAccountError(t *testing.T) {
	unsupportedErr := &UnsupportedAccountError{Message: unsupportedMessage}

	tests := []struct {
		name   string
		err    error
		want   string
		wantOK bool
	}{
		{name: "wrapped unsupported account error", err: fmt.Errorf("authorization error: %w", unsupportedErr), want: unsupportedMessage, wantOK: true},
		{name: "token refresh rejection", err: fmt.Errorf("refresh: %w", &oauth2.RetrieveError{ErrorCode: unsupportedAccountErrorCode, ErrorDescription: unsupportedMessage}), want: unsupportedMessage, wantOK: true},
		{name: "token refresh rejection without description", err: &oauth2.RetrieveError{ErrorCode: unsupportedAccountErrorCode}, want: unsupportedAccountFallback, wantOK: true},
		{name: "browser login callback rejection", err: fmt.Errorf("authorization error: %w", errors.New("authorization error from server: cli_unsupported_account "+unsupportedMessage)), want: unsupportedMessage, wantOK: true},
		{name: "browser login callback rejection without description", err: errors.New("authorization error: authorization error from server: cli_unsupported_account "), want: unsupportedAccountFallback, wantOK: true},
		{name: "other browser login callback error", err: errors.New("authorization error: authorization error from server: access_denied denied")},
		{name: "other retrieve error", err: &oauth2.RetrieveError{ErrorCode: "invalid_grant", ErrorDescription: "expired"}},
		{name: "other error", err: errors.New("boom")},
		{name: "nil error", err: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := AsUnsupportedAccountError(tt.err)
			if ok != tt.wantOK {
				t.Fatalf("expected ok=%v, got %v", tt.wantOK, ok)
			}
			if !ok {
				return
			}
			if got.Message != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got.Message)
			}
		})
	}
}
