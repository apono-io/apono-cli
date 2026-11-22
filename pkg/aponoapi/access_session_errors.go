package aponoapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode"
)

const (
	AccessSessionReasonNotActive = "not_active"
	AccessSessionReasonNotFound  = "not_found"
	AccessSessionReasonUnknown   = "unknown"
)

const (
	FailureFieldHTTPStatus = "http_status"
	FailureFieldReason     = "reason"
)

const serverMessageMaxLength = 500

type AccessSessionUnavailableError struct {
	Reason      string
	SessionID   string
	SessionName string
	HTTPStatus  int
}

func (e *AccessSessionUnavailableError) Error() string {
	switch e.Reason {
	case AccessSessionReasonNotActive:
		sessionLabel := e.SessionName
		if sessionLabel == "" {
			sessionLabel = e.SessionID
		}
		return fmt.Sprintf("You don't have active access to %s. Request it again from the Apono portal or with \"apono request create\".", sessionLabel)
	case AccessSessionReasonNotFound:
		return fmt.Sprintf("Access session %s was not found.", e.SessionID)
	default:
		return fmt.Sprintf("Access details for session %s are not available right now. Try again, or request access again if you no longer have it.", e.SessionID)
	}
}

type APIError struct {
	HTTPStatus int
	Message    string
}

func (e *APIError) Error() string {
	return e.Message
}

type responseBodyCarrier interface {
	Body() []byte
}

func DescribeAPIError(err error, resp *http.Response) error {
	if err == nil {
		return nil
	}

	if _, isUnsupportedAccount := AsUnsupportedAccountError(err); isUnsupportedAccount {
		return err
	}

	if resp == nil {
		return &APIError{Message: err.Error()}
	}

	if resp.StatusCode < http.StatusMultipleChoices {
		return &APIError{HTTPStatus: resp.StatusCode, Message: err.Error()}
	}

	message := resp.Status
	if serverMessage := serverMessageFromError(err); serverMessage != "" {
		message = fmt.Sprintf("%s: %s", resp.Status, serverMessage)
	}

	return &APIError{HTTPStatus: resp.StatusCode, Message: message}
}

func FailureFields(err error) (httpStatus, reason string) {
	var unavailableErr *AccessSessionUnavailableError
	if errors.As(err, &unavailableErr) {
		return formatHTTPStatus(unavailableErr.HTTPStatus), unavailableErr.Reason
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return formatHTTPStatus(apiErr.HTTPStatus), ""
	}

	return "", ""
}

func serverMessageFromError(err error) string {
	var bodyCarrier responseBodyCarrier
	if !errors.As(err, &bodyCarrier) {
		return ""
	}

	var body struct {
		Message string `json:"message"`
	}
	if jsonErr := json.Unmarshal(bodyCarrier.Body(), &body); jsonErr != nil {
		return ""
	}

	return sanitizeServerMessage(body.Message)
}

func sanitizeServerMessage(message string) string {
	printable := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, message)

	runes := []rune(printable)
	if len(runes) > serverMessageMaxLength {
		return string(runes[:serverMessageMaxLength])
	}
	return printable
}

func formatHTTPStatus(status int) string {
	if status == 0 {
		return ""
	}
	return strconv.Itoa(status)
}
