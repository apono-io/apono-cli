package approval

import "strings"

// sqlKeywords are SQL command keywords used to extract the operation prefix
var sqlKeywords = []string{
	"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER",
	"TRUNCATE", "GRANT", "REVOKE", "EXEC", "EXECUTE", "MERGE", "REPLACE",
	"SHOW", "DESCRIBE", "EXPLAIN", "WITH", "SET", "USE", "BEGIN", "COMMIT",
	"ROLLBACK", "CALL",
}

// sqlArgKeys are argument keys that commonly contain SQL statements
var sqlArgKeys = []string{"sql", "query", "statement"}

// ExtractSuggestedPattern builds a pattern string from tool name and SQL arguments.
// For example, tool "query" with SQL "CREATE TABLE users..." returns "query:CREATE*".
// If no SQL is found, returns "toolName:*".
func ExtractSuggestedPattern(toolName string, args map[string]interface{}) string {
	// Check well-known SQL argument keys first
	for _, key := range sqlArgKeys {
		if val, ok := args[key].(string); ok {
			if prefix := extractSQLPrefix(val); prefix != "" {
				return toolName + ":" + prefix + "*"
			}
		}
	}

	// Check all string values for SQL-like content
	for _, v := range args {
		if val, ok := v.(string); ok {
			if prefix := extractSQLPrefix(val); prefix != "" {
				return toolName + ":" + prefix + "*"
			}
		}
	}

	return toolName + ":*"
}

// extractSQLPrefix returns the first SQL keyword from a string, or "" if none found.
func extractSQLPrefix(s string) string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return ""
	}

	upper := strings.ToUpper(trimmed)
	for _, kw := range sqlKeywords {
		if strings.HasPrefix(upper, kw) {
			return kw
		}
	}

	return ""
}

// MatchesPattern checks if a tool call matches an approved pattern.
// Pattern format: "toolName:PREFIX*" where PREFIX is a SQL keyword.
// "toolName:*" matches all operations on that tool.
func MatchesPattern(pattern string, toolName string, args map[string]interface{}) bool {
	parts := strings.SplitN(pattern, ":", 2)
	if len(parts) != 2 {
		return false
	}

	patternTool := parts[0]
	patternPrefix := parts[1]

	// Tool name must match exactly
	if patternTool != toolName {
		return false
	}

	// Wildcard matches everything for this tool
	if patternPrefix == "*" {
		return true
	}

	// Remove trailing * for prefix matching
	prefix := strings.TrimSuffix(patternPrefix, "*")
	if prefix == "" {
		return true
	}

	// Extract the SQL prefix from args and compare
	actual := ExtractSuggestedPattern(toolName, args)
	// actual is "toolName:PREFIX*", extract just the prefix part
	actualParts := strings.SplitN(actual, ":", 2)
	if len(actualParts) != 2 {
		return false
	}
	actualPrefix := strings.TrimSuffix(actualParts[1], "*")

	return strings.HasPrefix(strings.ToUpper(actualPrefix), strings.ToUpper(prefix))
}
