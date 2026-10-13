package agent

import "testing"

func clearEnv(t *testing.T) {
	t.Helper()
	for _, a := range knownAgents {
		for _, v := range a.vars {
			t.Setenv(v, "")
		}
	}
	t.Setenv("AI_AGENT", "")
}

func TestDetect(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"none", nil, ""},
		{"claude code", map[string]string{"CLAUDECODE": "1"}, "claude-code"},
		{"cursor", map[string]string{"CURSOR_TRACE_ID": "abc"}, "cursor"},
		{"gemini", map[string]string{"GEMINI_CLI": "1"}, "gemini-cli"},
		{"copilot", map[string]string{"COPILOT_MODEL": "gpt"}, "github-copilot"},
		{"known agent wins over AI_AGENT", map[string]string{"CLAUDECODE": "1", "AI_AGENT": "claude-code_2-1-237_agent"}, "claude-code"},
		{"unknown agent falls back to sanitized AI_AGENT", map[string]string{"AI_AGENT": "Some Agent!/v2"}, "someagentv2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			if got := Detect(); got != tt.want {
				t.Errorf("Detect() = %q, want %q", got, tt.want)
			}
		})
	}
}
