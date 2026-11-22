package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
)

func TestIsFailedSessionFetch(t *testing.T) {
	unavailableErr := &aponoapi.AccessSessionUnavailableError{
		Reason:    aponoapi.AccessSessionReasonNotFound,
		SessionID: "session-1",
	}
	apiErr := &aponoapi.APIError{HTTPStatus: 500, Message: "500 Internal Server Error: boom"}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "unavailable session", err: unavailableErr, want: true},
		{name: "api error", err: apiErr, want: true},
		{name: "wrapped unavailable session", err: fmt.Errorf("could not fetch session details: %w", unavailableErr), want: true},
		{name: "wrapped api error", err: fmt.Errorf("could not fetch session details: %w", apiErr), want: true},
		{name: "plain error", err: errors.New("unknown flag: --nope"), want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isFailedSessionFetch(tt.err); got != tt.want {
				t.Fatalf("isFailedSessionFetch(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}
