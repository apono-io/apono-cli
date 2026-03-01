package approval

import "testing"

func TestExtractSuggestedPattern(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		args     map[string]interface{}
		want     string
	}{
		{
			name:     "CREATE TABLE SQL",
			toolName: "query",
			args:     map[string]interface{}{"sql": "CREATE TABLE users (id INT)"},
			want:     "query:CREATE*",
		},
		{
			name:     "DROP TABLE SQL",
			toolName: "execute",
			args:     map[string]interface{}{"sql": "DROP TABLE users"},
			want:     "execute:DROP*",
		},
		{
			name:     "INSERT INTO SQL",
			toolName: "query",
			args:     map[string]interface{}{"query": "INSERT INTO users VALUES (1)"},
			want:     "query:INSERT*",
		},
		{
			name:     "no SQL in args",
			toolName: "run",
			args:     map[string]interface{}{"command": "ls -la"},
			want:     "run:*",
		},
		{
			name:     "empty args",
			toolName: "query",
			args:     map[string]interface{}{},
			want:     "query:*",
		},
		{
			name:     "lowercase SQL",
			toolName: "query",
			args:     map[string]interface{}{"sql": "create table users (id int)"},
			want:     "query:CREATE*",
		},
		{
			name:     "SQL with leading whitespace",
			toolName: "query",
			args:     map[string]interface{}{"sql": "  ALTER TABLE users ADD COLUMN name TEXT"},
			want:     "query:ALTER*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractSuggestedPattern(tt.toolName, tt.args)
			if got != tt.want {
				t.Errorf("ExtractSuggestedPattern(%q, %v) = %q, want %q", tt.toolName, tt.args, got, tt.want)
			}
		})
	}
}

func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		toolName string
		args     map[string]interface{}
		want     bool
	}{
		{
			name:     "exact prefix match",
			pattern:  "query:CREATE*",
			toolName: "query",
			args:     map[string]interface{}{"sql": "CREATE TABLE users (id INT)"},
			want:     true,
		},
		{
			name:     "different SQL command",
			pattern:  "query:CREATE*",
			toolName: "query",
			args:     map[string]interface{}{"sql": "DROP TABLE users"},
			want:     false,
		},
		{
			name:     "different tool name",
			pattern:  "query:CREATE*",
			toolName: "execute",
			args:     map[string]interface{}{"sql": "CREATE TABLE users (id INT)"},
			want:     false,
		},
		{
			name:     "wildcard only pattern",
			pattern:  "query:*",
			toolName: "query",
			args:     map[string]interface{}{"sql": "DROP TABLE users"},
			want:     true,
		},
		{
			name:     "wildcard different tool",
			pattern:  "query:*",
			toolName: "execute",
			args:     map[string]interface{}{"sql": "DROP TABLE users"},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchesPattern(tt.pattern, tt.toolName, tt.args)
			if got != tt.want {
				t.Errorf("MatchesPattern(%q, %q, %v) = %v, want %v", tt.pattern, tt.toolName, tt.args, got, tt.want)
			}
		})
	}
}
