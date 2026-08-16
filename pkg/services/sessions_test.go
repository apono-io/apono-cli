package services

import (
	"testing"

	"github.com/apono-io/apono-cli/pkg/clientapi"
)

func TestShouldSuggestCredentialsReset(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		canReset bool
		want     bool
	}{
		{name: "not collected yet", status: "new", canReset: true, want: false},
		{name: "status casing is ignored", status: "NEW", canReset: true, want: false},
		{name: "already collected", status: "stale", canReset: true, want: true},
		{name: "unavailable", status: "unavailable", canReset: true, want: true},
		{name: "collected but reset not allowed", status: "stale", canReset: false, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &clientapi.AccessSessionClientModel{
				Credentials: *clientapi.NewNullableAccessSessionClientModelCredentials(
					clientapi.NewAccessSessionClientModelCredentials("cred-id", tt.status, tt.canReset),
				),
			}

			if got := ShouldSuggestCredentialsReset(session); got != tt.want {
				t.Errorf("ShouldSuggestCredentialsReset(%q, canReset=%v) = %v, want %v", tt.status, tt.canReset, got, tt.want)
			}
		})
	}
}

func TestShouldSuggestCredentialsReset_sessionWithoutCredentials(t *testing.T) {
	if ShouldSuggestCredentialsReset(&clientapi.AccessSessionClientModel{}) {
		t.Error("expected no suggestion for a session that does not use credentials")
	}
}
