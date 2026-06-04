package logshipping

import "testing"

func TestWithinCap(t *testing.T) {
	cases := []struct {
		count int32
		want  bool
	}{
		{1, true},
		{maxEventsPerInvocation, true},
		{maxEventsPerInvocation + 1, false},
		{maxEventsPerInvocation + 100, false},
	}
	for _, tc := range cases {
		if got := withinCap(tc.count); got != tc.want {
			t.Errorf("withinCap(%d) = %v, want %v", tc.count, got, tc.want)
		}
	}
}
