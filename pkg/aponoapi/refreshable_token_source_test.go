package aponoapi

import (
	"errors"
	"fmt"
	"net/url"
	"testing"

	"golang.org/x/oauth2"
)

func TestIsInvalidGrant(t *testing.T) {
	invalidGrant := &oauth2.RetrieveError{Body: []byte(`{"error":"invalid_grant"}`)}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "unrelated error",
			err:  errors.New("connection refused"),
			want: false,
		},
		{
			name: "retrieve error without invalid_grant",
			err:  &oauth2.RetrieveError{Body: []byte(`{"error":"invalid_client"}`)},
			want: false,
		},
		{
			name: "bare invalid_grant",
			err:  invalidGrant,
			want: true,
		},
		{
			name: "invalid_grant wrapped by net/http and fmt",
			err:  fmt.Errorf("get session: %w", &url.Error{Op: "Get", URL: "https://api.apono.io", Err: invalidGrant}),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInvalidGrant(tt.err); got != tt.want {
				t.Errorf("IsInvalidGrant() = %v, want %v", got, tt.want)
			}
		})
	}
}
