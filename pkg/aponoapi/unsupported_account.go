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
)

// UnsupportedAccountError is returned when the account can't be used from the CLI
// (e.g. 1Password Privileged Access accounts). Message is the server's user-facing text.
type UnsupportedAccountError struct {
	Message string
}

func (e *UnsupportedAccountError) Error() string {
	return e.Message
}

// UserFacingError replaces errors that carry a server message the user should see as-is.
func UserFacingError(err error) error {
	var unsupportedErr *UnsupportedAccountError
	if errors.As(err, &unsupportedErr) {
		return unsupportedErr
	}

	var retrieveErr *oauth2.RetrieveError
	if errors.As(err, &retrieveErr) && retrieveErr.ErrorCode == unsupportedAccountErrorCode && retrieveErr.ErrorDescription != "" {
		return &UnsupportedAccountError{Message: retrieveErr.ErrorDescription}
	}

	return err
}

// withUnsupportedAccountCheck returns a copy of client whose responses tagged with
// cli_unsupported_account become an UnsupportedAccountError instead of a bare 403.
func withUnsupportedAccountCheck(client *http.Client) *http.Client {
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
		body.Message = "The Apono CLI doesn't support this account"
	}

	return nil, &UnsupportedAccountError{Message: body.Message}
}
