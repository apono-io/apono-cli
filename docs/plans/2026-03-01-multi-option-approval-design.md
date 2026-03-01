# Multi-Option Approval with Local Caching

## Problem

When an AI agent performs multiple similar operations (e.g., creating 10 database tables), each operation triggers a separate HITL approval request via Slack. The user must approve each one individually, which is slow and disruptive.

## Solution

Replace the binary approve/deny Slack flow with 4 response options, and add a CLI-side in-memory cache so that "approve intent" and "approve pattern" decisions auto-approve subsequent matching operations without hitting the backend.

## Approval Modes

| Mode | Slack Button | Behavior |
|------|-------------|----------|
| `approve_once` | "Approve Once" | Approve this single action only |
| `deny` | "Reject" | Block this action |
| `approve_intent` | "Approve Intent" | Approve all actions with the same intent string (session-scoped) |
| `approve_pattern` | "Approve Pattern: query:CREATE*" | Approve all actions matching this tool+SQL prefix pattern (session-scoped) |

Both `approve_intent` and `approve_pattern` are **temporary** — cached in-memory for the MCP server process lifetime only. Not persisted.

## Architecture

### Components

```
ExecuteDynamicTool (proxy_manager.go)
    |
    v
ApprovalCache (new: approval/cache.go)
    |
    +-- Check local cache (intents + patterns)
    |       |
    |       +-- HIT: return auto-approved
    |       |
    |       +-- MISS: delegate to AponoActionApprover
    |               |
    |               +-- Send to backend with intent + suggested_pattern
    |               +-- Slack shows 4 buttons
    |               +-- Poll for response (includes mode)
    |               +-- Return ApprovalResult with mode
    |
    +-- Cache result if mode is approve_intent or approve_pattern
    |
    v
Execute or block
```

### Type Changes

**`approval/types.go`** — New types and updated interface:

```go
type ApprovalMode string

const (
    ApprovalModeApproveOnce    ApprovalMode = "approve_once"
    ApprovalModeDeny           ApprovalMode = "deny"
    ApprovalModeApproveIntent  ApprovalMode = "approve_intent"
    ApprovalModeApprovePattern ApprovalMode = "approve_pattern"
)

type ApprovalResult struct {
    Approved bool
    Mode     ApprovalMode
    Pattern  string // set when Mode == ApprovalModeApprovePattern
}

// ApprovalRequest gets new field:
SuggestedPattern string // e.g., "query:CREATE*"

// Approver interface changes return type:
RequestApproval(ctx, req) (*ApprovalResult, error)

// ActionApprovalDecision gets new fields from backend:
Mode    string `json:"mode,omitempty"`    // "approve_once", "approve_intent", "approve_pattern"
Pattern string `json:"pattern,omitempty"` // the approved pattern
```

### Pattern Extraction

Before sending the approval request, extract a suggested pattern from the tool call:

1. Scan `args` for SQL content (keys: `sql`, `query`, `statement`, or any string value starting with a SQL keyword)
2. Extract the first SQL command: `CREATE TABLE users...` -> `CREATE`
3. Combine with tool name: `query:CREATE*`
4. Send as `suggested_pattern` in the API request params

The Slack button displays: **"Approve Pattern: query:CREATE*"**

### Approval Cache

New file: `approval/cache.go`

```go
type ApprovalCache struct {
    delegate         Approver                // real approver (AponoActionApprover)
    approvedIntents  map[string]bool         // intent -> approved
    approvedPatterns []string                // glob patterns like "query:CREATE*"
    mu               sync.RWMutex
}
```

**Cache check logic:**
1. If `req.Intent != ""` and intent is in `approvedIntents` -> auto-approve
2. Build the operation's pattern string (same as suggested pattern extraction)
3. For each cached pattern, check if it matches via prefix/glob -> auto-approve
4. Otherwise, cache miss -> delegate to real approver

**Pattern matching:** `query:CREATE*` matches any operation where tool name is `query` and SQL starts with `CREATE`. Simple prefix matching on the part before `*`.

### API Request Changes

The `Params` map in the approval request includes:
```json
{
  "name": "query",
  "arguments": {...},
  "intent": "Creating user management tables",
  "suggested_pattern": "query:CREATE*"
}
```

### API Response Changes

`ActionApprovalDecision` response includes:
```json
{
  "approved": true,
  "responder": "user@example.com",
  "mode": "approve_pattern",
  "pattern": "query:CREATE*"
}
```

When `mode` is `approve_once` or absent (backward compat), behavior is same as current approve. When `mode` is `approve_intent` or `approve_pattern`, the CLI caches the decision.

## Files to Modify

| File | Change |
|------|--------|
| `approval/types.go` | Add `ApprovalMode`, `ApprovalResult`, `SuggestedPattern` field, update `Approver` interface |
| `approval/apono_approver.go` | Update `RequestApproval` return type, include `suggested_pattern` in params, parse mode from response |
| `approval/cache.go` | **New file**: `ApprovalCache` with intent/pattern caching |
| `approval/pattern.go` | **New file**: `ExtractSuggestedPattern` and pattern matching |
| `proxy/proxy_manager.go` | Extract pattern, use `ApprovalCache`, handle `ApprovalResult` |
| `actions/mcp.go` | Wrap `AponoActionApprover` in `ApprovalCache` during init |
| `actions/test.go` | Update test command for new return type |

## Edge Cases

- **Empty intent**: No intent cache check, falls through to pattern check or backend
- **Empty pattern**: If no SQL found in args, suggested pattern is `toolName:*` (matches all ops on that tool)
- **Backend doesn't return mode**: Treat as `approve_once` (backward compatible)
- **Process restart**: Cache is lost, all approvals start fresh (by design — temporary)
