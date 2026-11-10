package urihandler

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var eventStamp = regexp.MustCompile(` at=[0-9]+(\.[0-9]+)? inv=([0-9]+-[0-9]+)$`)

// stampOf returns the click id a handler line ends with, or "" if the line has
// no at=/inv= suffix.
func stampOf(message string) string {
	m := eventStamp.FindStringSubmatch(message)
	if m == nil {
		return ""
	}
	return m[2]
}

// runHandlerScript executes the embedded handler.sh with zsh against an isolated
// HOME and a PATH without apono, and returns the exit code and drained lines.
func runHandlerScript(t *testing.T, uri string) (int, []LogLine) {
	t.Helper()
	zsh, err := exec.LookPath("zsh")
	if err != nil {
		t.Skip("zsh not available")
	}
	home := t.TempDir()
	logDir := filepath.Join(home, bundleParentDir)
	if err = os.MkdirAll(logDir, 0o700); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(t.TempDir(), "handler.sh")
	if err = os.WriteFile(script, []byte(handlerShellTemplate), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(zsh, "-f", script, uri) //nolint:gosec // test runs the embedded script
	cmd.Env = []string{"HOME=" + home, "PATH=/usr/bin:/bin"}
	code := 0
	if err = cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("run handler: %v", err)
		}
		code = exitErr.ExitCode()
	}

	lines, err := DrainLog(filepath.Join(logDir, handlerLogFileName))
	if err != nil {
		t.Fatal(err)
	}
	return code, lines
}

func TestHandlerScript_invalidURL_logsReasonAndShape(t *testing.T) {
	cases := []struct {
		name      string
		uri       string
		wantParts []string
		forbidden []string
	}{
		{"missing uri", "", []string{"reason=missing_uri"}, nil},
		{
			"unsupported target",
			"https://example.com/?x=1",
			[]string{"reason=unsupported_target", "target=https://example.com has_userinfo=no has_path=yes", "x=<redacted>"},
			[]string{"x=1"},
		},
		{
			"target without scheme is redacted",
			"garbage-SECRET?x=1",
			[]string{"reason=unsupported_target", "target=<redacted>"},
			[]string{"SECRET"},
		},
		{
			"userinfo before the query is dropped",
			"apono://alice-SECRET:p4ssw0rd@connect?session=s",
			[]string{"reason=unsupported_target", "target=apono://connect has_userinfo=yes has_path=no"},
			[]string{"p4ssw0rd", "alice", "SECRET"},
		},
		{
			"path before the query is dropped",
			"apono://connect/SECRET-PATH?session=s",
			[]string{"reason=unsupported_target", "target=apono://connect has_userinfo=no has_path=yes"},
			[]string{"SECRET"},
		},
		{
			"host that is not a plain name is redacted",
			"apono://SECRET host<x>?session=s",
			[]string{"reason=unsupported_target", "target=apono://<redacted>"},
			[]string{"SECRET"},
		},
		{
			"parameter without '=' is not logged as a name",
			"apono://connect?eyJhbGciOiJIUzI1NiJ9.SECRET&session=s",
			[]string{"reason=missing_params", "params=[<redacted>=<redacted>,session=s]"},
			[]string{"SECRET", "eyJ"},
		},
		{
			"plain unknown parameter name is kept",
			"apono://connect?session=s&my_flag-1=SECRET",
			[]string{"reason=missing_params", "my_flag-1=<redacted>"},
			[]string{"SECRET"},
		},
		{"missing params", "apono://connect?session=a%2Fb&account=acc", []string{"reason=missing_params", "session=a%2Fb", "account=acc"}, nil},
		{
			"encoded account", "apono://connect?session=s&account=a%20c&client=c&token=SECRET",
			[]string{"reason=encoded_account_client", "account=a%20c", "token=<redacted>"},
			[]string{"SECRET"},
		},
		{"control characters stripped", "apono://connect?session=x%0A&client=c\nERROR\tforged", []string{"reason=missing_params"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, lines := runHandlerScript(t, tc.uri)
			if code != 64 {
				t.Fatalf("exit code = %d, want 64", code)
			}
			if len(lines) != 2 {
				t.Fatalf("expected received + error lines, got %d: %+v", len(lines), lines)
			}
			received, failure := lines[0], lines[1]
			if !strings.HasPrefix(received.Message, "received launch request at=") {
				t.Errorf("received line = %+v", received)
			}
			if failure.Level != "ERROR" || !strings.HasPrefix(failure.Message, "invalid launch URL code=64 ") {
				t.Errorf("failure line = %+v", failure)
			}
			for _, l := range lines {
				if stampOf(l.Message) == "" {
					t.Errorf("line missing at=/inv= suffix: %+v", l)
				}
			}
			if stampOf(received.Message) != stampOf(failure.Message) {
				t.Errorf("lines of one click have different inv: %q vs %q", received.Message, failure.Message)
			}
			for _, want := range tc.wantParts {
				if !strings.Contains(failure.Message, want) {
					t.Errorf("failure message %q missing %q", failure.Message, want)
				}
			}
			for _, unwanted := range tc.forbidden {
				for _, l := range lines {
					if strings.Contains(l.Message, unwanted) {
						t.Errorf("handler log line %q must not contain %q", l.Message, unwanted)
					}
				}
			}
		})
	}
}

func TestHandlerScript_encodedSession_passesParsingStep(t *testing.T) {
	code, lines := runHandlerScript(t, "apono://connect?session=aws-rds%2Fcluster&account=acc&client=cli")
	if code != 127 { // parsing succeeded; apono is deliberately not on PATH
		t.Fatalf("exit code = %d, want 127", code)
	}
	var parsed bool
	for _, l := range lines {
		if strings.HasPrefix(l.Message, "parsed launch params session=aws-rds%2Fcluster ") &&
			strings.Contains(l.Message, "session_has_pct=yes") {
			parsed = true
		}
		if strings.HasPrefix(l.Message, "invalid launch URL") {
			t.Errorf("encoded session must not be rejected: %+v", l)
		}
	}
	if !parsed {
		t.Errorf("expected a parsed launch params line, got %+v", lines)
	}
}
