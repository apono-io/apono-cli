package agent

import "testing"

// allVars is every env var Detect reads; tests clear them all so the
// environment running the tests (possibly an agent itself) can't leak in.
var allVars = []string{
	"CURSOR_TRACE_ID", "CURSOR_AGENT", "GEMINI_CLI",
	"CODEX_SANDBOX", "CODEX_CI", "CODEX_THREAD_ID",
	"ANTIGRAVITY_AGENT", "AUGMENT_AGENT", "OPENCODE_CLIENT",
	"CLAUDECODE", "CLAUDE_CODE", "REPL_ID",
	"COPILOT_MODEL", "COPILOT_ALLOW_ALL", "COPILOT_GITHUB_TOKEN",
	"AI_AGENT",
}

func clearEnv(t *testing.T) {
	t.Helper()
	for _, v := range allVars {
		t.Setenv(v, "")
	}
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
