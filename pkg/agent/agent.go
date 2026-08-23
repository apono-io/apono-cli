// Package agent detects whether the CLI was invoked by an AI coding agent.
package agent

import (
	"os"
	"strings"

	"github.com/mattn/go-isatty"
)

// knownAgents maps env vars set by each AI coding agent to a canonical name.
// Ported from https://github.com/vercel/vercel/tree/main/packages/detect-agent
// ponytail: env-var table only, no parent-process walk — covers all mainstream agents.
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

// Detect returns the canonical name of the AI agent that invoked the CLI,
// or "" when none is detected. Known agents win over the generic AI_AGENT
// value to keep names stable (AI_AGENT often embeds a version).
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

// IsInteractive reports whether stdout is attached to a terminal.
// False for agents, pipes, and scripts alike.
func IsInteractive() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}

// sanitize makes an arbitrary env value safe for HTTP headers and analytics:
// lowercase, [a-z0-9._-] only, max 64 chars.
func sanitize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if len(out) > 64 {
		out = out[:64]
	}
	return out
}
