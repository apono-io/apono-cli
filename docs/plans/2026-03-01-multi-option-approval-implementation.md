# Multi-Option Approval Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace binary approve/deny HITL flow with 4 options (approve once, reject, approve intent, approve pattern) and add CLI-side in-memory caching so repeated similar operations auto-approve without hitting the backend.

**Architecture:** An `ApprovalCache` wraps the real `AponoActionApprover`, checking local intent/pattern caches before delegating to the backend. The backend's Slack UI shows 4 buttons. The response includes a `mode` field that the CLI uses to cache intent or pattern decisions for the session.

**Tech Stack:** Go, MCP JSON-RPC, Apono action-approval REST API

---

### Task 1: Update Approval Types

**Files:**
- Modify: `pkg/commands/mcp/approval/types.go`

**Step 1: Add ApprovalMode and ApprovalResult types, update ApprovalRequest and Approver interface**

Replace the entire file with:

```go
package approval

import "context"

// ApprovalMode represents the type of approval decision
type ApprovalMode string

const (
	ApprovalModeApproveOnce    ApprovalMode = "approve_once"
	ApprovalModeDeny           ApprovalMode = "deny"
	ApprovalModeApproveIntent  ApprovalMode = "approve_intent"
	ApprovalModeApprovePattern ApprovalMode = "approve_pattern"
)

// ApprovalResult represents the outcome of an approval request
type ApprovalResult struct {
	Approved bool
	Mode     ApprovalMode
	Pattern  string // set when Mode == ApprovalModeApprovePattern
}

// ApprovalRequest represents a request for human approval of a risky operation
type ApprovalRequest struct {
	ToolName         string                 `json:"tool_name"`
	Arguments        map[string]interface{} `json:"arguments"`
	Reason           string                 `json:"reason"`
	RiskLevel        string                 `json:"risk_level"`
	MatchedRule      string                 `json:"matched_rule"`
	TargetID         string                 `json:"target_id"`
	IntegrationID    string                 `json:"integration_id"`
	Intent           string                 `json:"intent,omitempty"`
	SuggestedPattern string                 `json:"suggested_pattern,omitempty"`
}

// Approver requests approval for risky operations
type Approver interface {
	// RequestApproval submits an approval request and blocks until a decision is made or timeout
	RequestApproval(ctx context.Context, req ApprovalRequest) (*ApprovalResult, error)
}
```

**Step 2: Verify it compiles (it won't — callers need updating, that's expected)**

Run: `go build ./pkg/commands/mcp/approval/...`
Expected: Compiles (this package alone is fine, callers break later)

---

### Task 2: Create Pattern Extraction

**Files:**
- Create: `pkg/commands/mcp/approval/pattern.go`
- Create: `pkg/commands/mcp/approval/pattern_test.go`

**Step 1: Write tests for pattern extraction and matching**

```go
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
```

**Step 2: Run tests to verify they fail**

Run: `go test ./pkg/commands/mcp/approval/ -run TestExtract -v`
Expected: FAIL — functions not defined

**Step 3: Implement pattern extraction and matching**

```go
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
```

**Step 4: Run tests to verify they pass**

Run: `go test ./pkg/commands/mcp/approval/ -run "TestExtract|TestMatches" -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add pkg/commands/mcp/approval/pattern.go pkg/commands/mcp/approval/pattern_test.go
git commit -m "feat(mcp): add pattern extraction and matching for approval"
```

---

### Task 3: Create Approval Cache

**Files:**
- Create: `pkg/commands/mcp/approval/cache.go`
- Create: `pkg/commands/mcp/approval/cache_test.go`

**Step 1: Write tests for approval cache**

```go
package approval

import (
	"context"
	"testing"
)

// mockApprover is a test double that records calls and returns configured results
type mockApprover struct {
	calls  []ApprovalRequest
	result *ApprovalResult
	err    error
}

func (m *mockApprover) RequestApproval(_ context.Context, req ApprovalRequest) (*ApprovalResult, error) {
	m.calls = append(m.calls, req)
	return m.result, m.err
}

func TestApprovalCache_CacheMiss_DelegatesToApprover(t *testing.T) {
	mock := &mockApprover{
		result: &ApprovalResult{Approved: true, Mode: ApprovalModeApproveOnce},
	}
	cache := NewApprovalCache(mock)

	result, err := cache.RequestApproval(context.Background(), ApprovalRequest{
		ToolName: "query",
		Intent:   "create tables",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Approved {
		t.Fatal("expected approved")
	}
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 delegate call, got %d", len(mock.calls))
	}
}

func TestApprovalCache_ApproveIntent_CachesAndAutoApproves(t *testing.T) {
	mock := &mockApprover{
		result: &ApprovalResult{Approved: true, Mode: ApprovalModeApproveIntent},
	}
	cache := NewApprovalCache(mock)
	ctx := context.Background()

	// First call: delegates to approver
	_, err := cache.RequestApproval(ctx, ApprovalRequest{
		ToolName: "query",
		Intent:   "create tables",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second call with same intent: should auto-approve without delegating
	result, err := cache.RequestApproval(ctx, ApprovalRequest{
		ToolName: "query",
		Intent:   "create tables",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Approved {
		t.Fatal("expected auto-approved")
	}
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 delegate call (cached), got %d", len(mock.calls))
	}
}

func TestApprovalCache_ApprovePattern_CachesAndAutoApproves(t *testing.T) {
	mock := &mockApprover{
		result: &ApprovalResult{Approved: true, Mode: ApprovalModeApprovePattern, Pattern: "query:CREATE*"},
	}
	cache := NewApprovalCache(mock)
	ctx := context.Background()

	// First call: delegates to approver
	_, err := cache.RequestApproval(ctx, ApprovalRequest{
		ToolName:  "query",
		Arguments: map[string]interface{}{"sql": "CREATE TABLE users (id INT)"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second call with different SQL but same prefix: should auto-approve
	result, err := cache.RequestApproval(ctx, ApprovalRequest{
		ToolName:  "query",
		Arguments: map[string]interface{}{"sql": "CREATE TABLE orders (id INT)"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Approved {
		t.Fatal("expected auto-approved via pattern")
	}
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 delegate call (cached), got %d", len(mock.calls))
	}
}

func TestApprovalCache_ApprovePattern_NoMatchDelegates(t *testing.T) {
	callCount := 0
	mock := &mockApprover{
		result: &ApprovalResult{Approved: true, Mode: ApprovalModeApprovePattern, Pattern: "query:CREATE*"},
	}
	cache := NewApprovalCache(mock)
	ctx := context.Background()

	// First call: CREATE TABLE
	_, _ = cache.RequestApproval(ctx, ApprovalRequest{
		ToolName:  "query",
		Arguments: map[string]interface{}{"sql": "CREATE TABLE users (id INT)"},
	})
	callCount++

	// Second call: DROP TABLE — should NOT match CREATE* pattern, delegates to approver
	mock.result = &ApprovalResult{Approved: false, Mode: ApprovalModeDeny}
	_, _ = cache.RequestApproval(ctx, ApprovalRequest{
		ToolName:  "query",
		Arguments: map[string]interface{}{"sql": "DROP TABLE users"},
	})
	callCount++

	if len(mock.calls) != callCount {
		t.Fatalf("expected %d delegate calls (pattern mismatch), got %d", callCount, len(mock.calls))
	}
}

func TestApprovalCache_DenyDoesNotCache(t *testing.T) {
	mock := &mockApprover{
		result: &ApprovalResult{Approved: false, Mode: ApprovalModeDeny},
	}
	cache := NewApprovalCache(mock)
	ctx := context.Background()

	// First call: denied
	_, _ = cache.RequestApproval(ctx, ApprovalRequest{
		ToolName: "query",
		Intent:   "drop tables",
	})

	// Second call with same intent: should still delegate (deny is not cached)
	mock.result = &ApprovalResult{Approved: true, Mode: ApprovalModeApproveOnce}
	_, _ = cache.RequestApproval(ctx, ApprovalRequest{
		ToolName: "query",
		Intent:   "drop tables",
	})

	if len(mock.calls) != 2 {
		t.Fatalf("expected 2 delegate calls (deny not cached), got %d", len(mock.calls))
	}
}

func TestApprovalCache_EmptyIntent_NoIntentCacheHit(t *testing.T) {
	mock := &mockApprover{
		result: &ApprovalResult{Approved: true, Mode: ApprovalModeApproveIntent},
	}
	cache := NewApprovalCache(mock)
	ctx := context.Background()

	// Call with empty intent
	_, _ = cache.RequestApproval(ctx, ApprovalRequest{
		ToolName: "query",
		Intent:   "",
	})

	// Second call with empty intent: should still delegate (empty intent not cached)
	_, _ = cache.RequestApproval(ctx, ApprovalRequest{
		ToolName: "query",
		Intent:   "",
	})

	if len(mock.calls) != 2 {
		t.Fatalf("expected 2 delegate calls (empty intent), got %d", len(mock.calls))
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./pkg/commands/mcp/approval/ -run TestApprovalCache -v`
Expected: FAIL — `NewApprovalCache` not defined

**Step 3: Implement ApprovalCache**

```go
package approval

import (
	"context"
	"sync"

	"github.com/apono-io/apono-cli/pkg/utils"
)

// ApprovalCache wraps an Approver with in-memory caching for approved intents and patterns.
// Cache is session-scoped (lives for the MCP server process lifetime).
type ApprovalCache struct {
	delegate         Approver
	approvedIntents  map[string]bool
	approvedPatterns []string
	mu               sync.RWMutex
}

// NewApprovalCache creates a new cache wrapping the given approver.
func NewApprovalCache(delegate Approver) *ApprovalCache {
	return &ApprovalCache{
		delegate:        delegate,
		approvedIntents: make(map[string]bool),
	}
}

// RequestApproval checks the local cache first. On cache hit, returns auto-approved.
// On cache miss, delegates to the real approver and caches the result if applicable.
func (c *ApprovalCache) RequestApproval(ctx context.Context, req ApprovalRequest) (*ApprovalResult, error) {
	// Check cache
	if c.matchesCache(req) {
		utils.McpLogf("[ApprovalCache] Auto-approved from cache (tool=%s)", req.ToolName)
		return &ApprovalResult{Approved: true, Mode: ApprovalModeApproveOnce}, nil
	}

	// Cache miss — delegate to real approver
	result, err := c.delegate.RequestApproval(ctx, req)
	if err != nil {
		return nil, err
	}

	// Cache the decision if applicable
	if result.Approved {
		c.cacheResult(req, result)
	}

	return result, nil
}

// matchesCache checks if the request matches any cached intent or pattern.
func (c *ApprovalCache) matchesCache(req ApprovalRequest) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check intent cache
	if req.Intent != "" && c.approvedIntents[req.Intent] {
		utils.McpLogf("[ApprovalCache] Intent cache hit: %q", req.Intent)
		return true
	}

	// Check pattern cache
	for _, pattern := range c.approvedPatterns {
		if MatchesPattern(pattern, req.ToolName, req.Arguments) {
			utils.McpLogf("[ApprovalCache] Pattern cache hit: %q", pattern)
			return true
		}
	}

	return false
}

// cacheResult stores the approval decision based on mode.
func (c *ApprovalCache) cacheResult(req ApprovalRequest, result *ApprovalResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch result.Mode {
	case ApprovalModeApproveIntent:
		if req.Intent != "" {
			c.approvedIntents[req.Intent] = true
			utils.McpLogf("[ApprovalCache] Cached approved intent: %q", req.Intent)
		}
	case ApprovalModeApprovePattern:
		if result.Pattern != "" {
			c.approvedPatterns = append(c.approvedPatterns, result.Pattern)
			utils.McpLogf("[ApprovalCache] Cached approved pattern: %q", result.Pattern)
		}
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./pkg/commands/mcp/approval/ -run TestApprovalCache -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add pkg/commands/mcp/approval/cache.go pkg/commands/mcp/approval/cache_test.go
git commit -m "feat(mcp): add approval cache with intent and pattern support"
```

---

### Task 4: Update AponoActionApprover

**Files:**
- Modify: `pkg/commands/mcp/approval/apono_approver.go`

**Step 1: Update return types and add suggested_pattern + mode parsing**

Key changes:
1. `RequestApproval` returns `(*ApprovalResult, error)` instead of `(bool, error)`
2. Include `suggested_pattern` in the `Params` map
3. `ActionApprovalDecision` gets `Mode` and `Pattern` fields
4. Parse the mode from poll responses and map to `ApprovalResult`
5. Handle backward compat: if `Mode` is empty, default to `approve_once`

Update `ActionApprovalDecision`:
```go
type ActionApprovalDecision struct {
	Approved  bool     `json:"approved"`
	Responder string   `json:"responder"`
	Timestamp UnixTime `json:"timestamp"`
	Comment   *string  `json:"comment,omitempty"`
	Mode      string   `json:"mode,omitempty"`    // "approve_once", "approve_intent", "approve_pattern"
	Pattern   string   `json:"pattern,omitempty"` // pattern when mode is approve_pattern
}
```

Update `RequestApproval` signature and body:
```go
func (a *AponoActionApprover) RequestApproval(ctx context.Context, req ApprovalRequest) (*ApprovalResult, error) {
```

In the `Params` map, add:
```go
"suggested_pattern": req.SuggestedPattern,
```

Replace the auto-approval check after `createApproval`:
```go
if createResp != nil {
    switch createResp.Status {
    case StatusApproved:
        utils.McpLogf("[Approver] Auto-approved on create: id=%s", approvalID)
        return a.buildResult(createResp), nil
    case StatusDenied:
        utils.McpLogf("[Approver] Auto-denied on create: id=%s", approvalID)
        return &ApprovalResult{Approved: false, Mode: ApprovalModeDeny}, nil
    }
}
```

Replace the poll response handling to return `*ApprovalResult`:
```go
case StatusApproved:
    responder := ""
    if response.Response != nil {
        responder = response.Response.Responder
    }
    utils.McpLogf("[Approver] Approved by %s!", responder)
    return a.buildResult(response), nil

case StatusDenied:
    responder := ""
    if response.Response != nil {
        responder = response.Response.Responder
    }
    utils.McpLogf("[Approver] Denied by %s!", responder)
    return &ApprovalResult{Approved: false, Mode: ApprovalModeDeny}, nil
```

Add helper method:
```go
// buildResult converts an API response to an ApprovalResult, handling backward compat
func (a *AponoActionApprover) buildResult(resp *ActionApprovalResponse) *ApprovalResult {
	result := &ApprovalResult{Approved: true, Mode: ApprovalModeApproveOnce}

	if resp.Response != nil {
		switch ApprovalMode(resp.Response.Mode) {
		case ApprovalModeApproveIntent:
			result.Mode = ApprovalModeApproveIntent
		case ApprovalModeApprovePattern:
			result.Mode = ApprovalModeApprovePattern
			result.Pattern = resp.Response.Pattern
		}
	}

	return result
}
```

Update error return sites (timeout, cancel) to return `(*ApprovalResult, error)`:
```go
return nil, fmt.Errorf("approval request timed out after %v", a.timeout)
return nil, fmt.Errorf("approval request cancelled: %w", ctx.Err())
```

**Step 2: Verify it compiles**

Run: `go build ./pkg/commands/mcp/approval/...`
Expected: PASS

**Step 3: Commit**

```bash
git add pkg/commands/mcp/approval/apono_approver.go
git commit -m "feat(mcp): update approver to return ApprovalResult with mode and pattern"
```

---

### Task 5: Update Proxy Manager

**Files:**
- Modify: `pkg/commands/mcp/proxy/proxy_manager.go:229-272`

**Step 1: Extract suggested pattern and handle ApprovalResult**

In `ExecuteDynamicTool`, after extracting intent (line 227) and before risk detection (line 229):

```go
// Extract suggested pattern for approval
suggestedPattern := approval.ExtractSuggestedPattern(toolName, args)
```

Update the `ApprovalRequest` construction to include `SuggestedPattern`:
```go
approvalResult, err := m.approver.RequestApproval(ctx, approval.ApprovalRequest{
    ToolName:         toolName,
    Arguments:        args,
    Reason:           riskResult.Reason,
    RiskLevel:        riskLevelToString(riskResult.Level),
    MatchedRule:      riskResult.MatchedRule,
    TargetID:         backendID,
    IntegrationID:    integrationID,
    Intent:           intent,
    SuggestedPattern: suggestedPattern,
})
```

Change the result handling from `approved bool` to `approvalResult *ApprovalResult`:
```go
if err != nil {
    return nil, fmt.Errorf("approval request failed: %w", err)
}
if !approvalResult.Approved {
    return ToolCallResult{
        Content: []ContentItem{
            {Type: "text", Text: fmt.Sprintf("Operation blocked: %s (approval denied)", riskResult.Reason)},
        },
        IsError: true,
    }, nil
}
```

**Step 2: Verify it compiles**

Run: `go build ./pkg/commands/mcp/proxy/...`
Expected: PASS

**Step 3: Commit**

```bash
git add pkg/commands/mcp/proxy/proxy_manager.go
git commit -m "feat(mcp): update proxy manager for multi-option approval"
```

---

### Task 6: Wire Up Cache in MCP Command

**Files:**
- Modify: `pkg/commands/mcp/actions/mcp.go:220-233`

**Step 1: Wrap AponoActionApprover in ApprovalCache**

In the `case "approve":` block (line 224-228), wrap the approver:

```go
case "approve":
    riskDetector = risk.NewPatternRiskDetector(risk.DefaultRiskConfig())
    baseApprover := approval.NewAponoActionApprover(apiBaseURL, apiCfg.HTTPClient, client.Session.UserID, 5*time.Minute)
    approver = approval.NewApprovalCache(baseApprover)
    utils.McpLogf("Risk action: approve (risky ops require Apono approval with caching)")
```

**Step 2: Verify it compiles**

Run: `go build ./pkg/commands/mcp/actions/...`
Expected: PASS

**Step 3: Commit**

```bash
git add pkg/commands/mcp/actions/mcp.go
git commit -m "feat(mcp): wire approval cache into MCP command"
```

---

### Task 7: Update Test Command

**Files:**
- Modify: `pkg/commands/mcp/actions/test.go:359-491`

**Step 1: Update testApproveFlow for new return type**

The `approver.RequestApproval` now returns `(*ApprovalResult, error)` instead of `(bool, error)`. Update the test command:

Change the result handling:
```go
result, err := approver.RequestApproval(ctx, req)
if err != nil {
    return fmt.Errorf("approval request failed: %w", err)
}

fmt.Println()
if result.Approved {
    fmt.Println("=== RESULT: APPROVED ===")
    fmt.Printf("Mode: %s\n", result.Mode)
    if result.Pattern != "" {
        fmt.Printf("Pattern: %s\n", result.Pattern)
    }
} else {
    fmt.Println("=== RESULT: DENIED ===")
}
```

Also add `--suggested-pattern` flag alongside the existing `--intent` flag:
```go
var suggestedPattern string
// in flag setup:
cmd.Flags().StringVar(&suggestedPattern, "suggested-pattern", "", "Suggested pattern for approval (e.g., query:CREATE*)")
// in the request:
SuggestedPattern: suggestedPattern,
```

**Step 2: Verify it compiles**

Run: `go build ./pkg/commands/mcp/actions/...`
Expected: PASS

**Step 3: Commit**

```bash
git add pkg/commands/mcp/actions/test.go
git commit -m "feat(mcp): update test command for multi-option approval"
```

---

### Task 8: Full Build and Lint

**Step 1: Build everything**

Run: `go build ./...`
Expected: PASS

**Step 2: Run tests**

Run: `go test ./pkg/commands/mcp/approval/ -v`
Expected: All PASS (pattern tests + cache tests)

**Step 3: Run vet**

Run: `go vet ./pkg/commands/mcp/...`
Expected: PASS

**Step 4: Final commit if any fixups needed**

---

### Task 9: Verification

**Manual E2E test flow:**

1. Start MCP proxy with `--risk-action approve`
2. Send a risky tool call (e.g., `CREATE TABLE`) — should trigger Slack approval
3. In Slack, click "Approve Pattern: query:CREATE*"
4. Send another `CREATE TABLE` — should auto-approve instantly (no Slack notification)
5. Send a `DROP TABLE` — should trigger a new Slack approval (different pattern)
6. Restart MCP proxy — cache should be empty, all approvals start fresh

**Test commands:**
```bash
# Test with intent
apono mcp test approve-flow --intent "create tables" --suggested-pattern "query:CREATE*"

# Test without intent (backward compat)
apono mcp test approve-flow
```
