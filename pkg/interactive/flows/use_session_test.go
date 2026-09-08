package flows

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/apono-io/apono-cli/pkg/clientapi"
)

func sessionWithCredentials(status string, canReset bool) *clientapi.AccessSessionClientModel {
	return &clientapi.AccessSessionClientModel{
		Id: "session-id",
		Credentials: *clientapi.NewNullableAccessSessionClientModelCredentials(
			clientapi.NewAccessSessionClientModelCredentials("cred-id", status, canReset),
		),
	}
}

func sessionWithoutCredentials() *clientapi.AccessSessionClientModel {
	return &clientapi.AccessSessionClientModel{Id: "session-id"}
}

func TestMaybePrintErrorConnectingSuggestion(t *testing.T) {
	tests := []struct {
		name        string
		session     *clientapi.AccessSessionClientModel
		wantMessage bool
	}{
		{name: "non-credential-based integration", session: sessionWithoutCredentials(), wantMessage: false},
		{name: "fresh credentials", session: sessionWithCredentials("new", true), wantMessage: false},
		{name: "reset not allowed", session: sessionWithCredentials("stale", false), wantMessage: false},
		{name: "stale, resettable credentials", session: sessionWithCredentials("stale", true), wantMessage: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			var out bytes.Buffer
			cmd.SetOut(&out)

			if err := MaybePrintErrorConnectingSuggestion(cmd, tt.session); err != nil {
				t.Fatalf("MaybePrintErrorConnectingSuggestion() error = %v", err)
			}

			gotMessage := strings.Contains(out.String(), "Reset credentials")
			if gotMessage != tt.wantMessage {
				t.Errorf("output = %q, wantMessage = %v", out.String(), tt.wantMessage)
			}
		})
	}
}

func TestMaybePrintResetCredentialsSuggestion(t *testing.T) {
	tests := []struct {
		name        string
		session     *clientapi.AccessSessionClientModel
		wantMessage bool
	}{
		{name: "non-credential-based integration", session: sessionWithoutCredentials(), wantMessage: false},
		{name: "stale, resettable credentials", session: sessionWithCredentials("stale", true), wantMessage: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			var out bytes.Buffer
			cmd.SetOut(&out)

			if err := maybePrintResetCredentialsSuggestion(cmd, tt.session); err != nil {
				t.Fatalf("maybePrintResetCredentialsSuggestion() error = %v", err)
			}

			gotMessage := strings.Contains(out.String(), "new set of credentials")
			if gotMessage != tt.wantMessage {
				t.Errorf("output = %q, wantMessage = %v", out.String(), tt.wantMessage)
			}
		})
	}
}
