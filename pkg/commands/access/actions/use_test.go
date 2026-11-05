package actions

import "testing"

func TestResolveSessionIDArg(t *testing.T) {
	tests := []struct {
		name       string
		arg        string
		urlEncoded bool
		want       string
		wantErr    bool
	}{
		{name: "plain id from terminal is untouched", arg: "aws-rds-mysql-abc123", want: "aws-rds-mysql-abc123"},
		{name: "terminal arg with percent is not decoded", arg: "a%2Fb", want: "a%2Fb"},
		{name: "plain id from launch URL", arg: "aws-rds-mysql-abc123", urlEncoded: true, want: "aws-rds-mysql-abc123"},
		{
			name:       "encoded slashes from launch URL are decoded once",
			arg:        "aws-rds-mysql-dev%2Fmysql%2Feu1-payments-dev-commondev-aurora-cluster-d4bfac",
			urlEncoded: true,
			want:       "aws-rds-mysql-dev/mysql/eu1-payments-dev-commondev-aurora-cluster-d4bfac",
		},
		{name: "lowercase hex is accepted", arg: "a%2fb", urlEncoded: true, want: "a/b"},
		{name: "double-encoded is decoded only once", arg: "a%252Fb", urlEncoded: true, want: "a%2Fb"},
		{name: "plus sign is preserved", arg: "a+b", urlEncoded: true, want: "a+b"},
		{name: "malformed escape", arg: "a%zzb", urlEncoded: true, wantErr: true},
		{name: "truncated escape", arg: "a%2", urlEncoded: true, wantErr: true},
		{name: "control character after decode", arg: "a%0Ab", urlEncoded: true, wantErr: true},
		{name: "NUL after decode", arg: "a%00b", urlEncoded: true, wantErr: true},
		{name: "DEL after decode", arg: "a%7Fb", urlEncoded: true, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveSessionIDArg(tt.arg, tt.urlEncoded)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
