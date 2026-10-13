package agent

import (
	"os"
	"strings"
)

var knownAgents = []struct {
	name string
	vars []string
}{
	{"cursor", []string{"CURSOR_TRACE_ID", "CURSOR_AGENT"}},
	{"gemini-cli", []string{"GEMINI_CLI"}},
	{"codex", []string{"CODEX_SANDBOX", "CODEX_CI", "CODEX_THREAD_ID"}},
	{"antigravity", []string{"ANTIGRAVITY_AGENT"}},
	{"augment", []string{"AUGMENT_AGENT"}},
	{"opencode", []string{"OPENCODE_CLIENT"}},
	{"claude-code", []string{"CLAUDECODE", "CLAUDE_CODE"}},
	{"replit", []string{"REPL_ID"}},
	{"github-copilot", []string{"COPILOT_MODEL", "COPILOT_ALLOW_ALL", "COPILOT_GITHUB_TOKEN"}},
}

func Detect() string {
	for _, a := range knownAgents {
		for _, v := range a.vars {
			if os.Getenv(v) != "" {
				return a.name
			}
		}
	}
	if _, err := os.Stat("/opt/.devin"); err == nil {
		return "devin"
	}
	if raw := os.Getenv("AI_AGENT"); raw != "" {
		return sanitize(raw)
	}
	return ""
}

func sanitize(s string) string {
	out := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			return r
		}
		return -1
	}, strings.ToLower(s))
	if len(out) > 64 {
		out = out[:64]
	}
	return out
}
