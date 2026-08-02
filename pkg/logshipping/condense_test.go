package logshipping

import (
	"strings"
	"testing"
)

func TestCollapseRepeats(t *testing.T) {
	const ssoLine = "The SSO session associated with this profile has expired or is otherwise invalid."

	cases := []struct {
		name string
		text string
		want string
	}{
		{
			name: "nothing repeats",
			text: "first\nsecond\nthird",
			want: "first\nsecond\nthird",
		},
		{
			name: "adjacent repeats collapse with a count",
			text: "boom\nboom\nboom",
			want: "boom [repeated 3 times]",
		},
		{
			name: "repeats separated by blank lines still collapse",
			text: "client exited with code 1\n\n" + ssoLine + "\n\n" + ssoLine + "\n\n" + ssoLine + "\n",
			want: "client exited with code 1\n" + ssoLine + " [repeated 3 times]",
		},
		{
			name: "a later run is counted separately",
			text: "a\na\nb\na\na",
			want: "a [repeated 2 times]\nb\na [repeated 2 times]",
		},
		{
			name: "empty text stays empty",
			text: "",
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := collapseRepeats(tc.text); got != tc.want {
				t.Errorf("collapseRepeats() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTruncate_underLimitIsUntouched(t *testing.T) {
	text := strings.Repeat("x", 10)
	if got := truncate(text, 10); got != text {
		t.Errorf("truncate() = %q, want it unchanged", got)
	}
}

func TestTruncate_overLimitMarksTheCut(t *testing.T) {
	got := truncate(strings.Repeat("x", 20), 10)

	if !strings.HasSuffix(got, truncationSuffix) {
		t.Errorf("truncate() = %q, want it to end with the truncation marker", got)
	}
	if body := strings.TrimSuffix(got, truncationSuffix); len(body) != 10 {
		t.Errorf("kept %d bytes, want 10", len(body))
	}
}

func TestTruncate_doesNotSplitARune(t *testing.T) {
	// Each 😀 is four bytes, so a limit of 10 lands inside the third one.
	got := truncate(strings.Repeat("😀", 5), 10)

	body := strings.TrimSuffix(got, truncationSuffix)
	if len(body) != 8 {
		t.Errorf("kept %d bytes, want 8 (two whole runes)", len(body))
	}
	if !strings.ContainsRune(body, '😀') || strings.ToValidUTF8(body, "?") != body {
		t.Errorf("truncate() produced invalid UTF-8: %q", body)
	}
}

func TestCondense_boundsARepeatingClientFailure(t *testing.T) {
	const ssoLine = "The SSO session associated with this profile has expired or is otherwise invalid."

	text := "client exited with code 1\n" + strings.Repeat("\n"+ssoLine, 200)
	got := condense(text)

	if len(got) > maxFieldBytes+len(truncationSuffix) {
		t.Errorf("condense() returned %d bytes, want it bounded by %d", len(got), maxFieldBytes)
	}
	if !strings.Contains(got, "client exited with code 1") {
		t.Errorf("condense() dropped the leading diagnostic: %q", got)
	}
	if !strings.Contains(got, "[repeated 200 times]") {
		t.Errorf("condense() did not collapse the repeats: %q", got)
	}
	if strings.Count(got, ssoLine) != 1 {
		t.Errorf("expected the repeated line to survive exactly once, got %d copies", strings.Count(got, ssoLine))
	}
}
