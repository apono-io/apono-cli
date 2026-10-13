package aponoapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

const (
	errorCodeHeader              = "X-Apono-Error-Code"
	unsupportedAccountErrorCode  = "cli_unsupported_account"
	unsupportedAccountMaxBodyLen = 64 * 1024
	unsupportedAccountFallback   = "The Apono CLI doesn't support this account. Use the Apono web portal to manage your access instead."
)

// UnsupportedAccountError is returned when the account can't be used from the CLI
// (e.g. 1Password Privileged Access accounts). Message is the server's user-facing text.
type UnsupportedAccountError struct {
	Message string
}

func (e *UnsupportedAccountError) Error() string {
	return e.Message
}

// AsUnsupportedAccountError reports whether err means the account can't be used from the CLI,
// either from an API response or from an OAuth token request the server rejected.
func AsUnsupportedAccountError(err error) (*UnsupportedAccountError, bool) {
	var unsupportedErr *UnsupportedAccountError
	if errors.As(err, &unsupportedErr) {
		return unsupportedErr, true
	}

	var retrieveErr *oauth2.RetrieveError
	if errors.As(err, &retrieveErr) && retrieveErr.ErrorCode == unsupportedAccountErrorCode {
		message := retrieveErr.ErrorDescription
		if message == "" {
			message = unsupportedAccountFallback
		}
		return &UnsupportedAccountError{Message: message}, true
	}

	return nil, false
}

// WithUnsupportedAccountCheck returns a copy of client whose responses tagged with
// cli_unsupported_account become an UnsupportedAccountError instead of a bare 403.
func WithUnsupportedAccountCheck(client *http.Client) *http.Client {
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}

	wrapped := *client
	wrapped.Transport = &unsupportedAccountTransport{base: base}
	return &wrapped
}

type unsupportedAccountTransport struct {
	base http.RoundTripper
}

func (t *unsupportedAccountTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil || resp.Header.Get(errorCodeHeader) != unsupportedAccountErrorCode {
		return resp, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var body struct {
		Message string `json:"message"`
	}
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, unsupportedAccountMaxBodyLen))
	if jsonErr := json.Unmarshal(bodyBytes, &body); jsonErr != nil || body.Message == "" {
		body.Message = unsupportedAccountFallback
	}

	return nil, &UnsupportedAccountError{Message: body.Message}
}
