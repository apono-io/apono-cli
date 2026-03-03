# Approval Notifications Design

## Goal

Show the user real-time messages in Claude Code when a risky operation triggers the approval flow:
1. "Waiting for approval..." when the approval request is sent
2. "Approved by X" or "Denied by X" when the decision arrives

## Approach: MCP Logging Notifications

Use the MCP spec's `notifications/message` method to send JSON-RPC notifications to stdout while the tool call is still in progress.

### Message format

```json
{"jsonrpc":"2.0","method":"notifications/message","params":{"level":"warning","data":"Risky operation detected. Waiting for approval in Apono..."}}
{"jsonrpc":"2.0","method":"notifications/message","params":{"level":"info","data":"Operation approved by admin@company.com"}}
```

## Changes

### 1. Add Notifier type to proxy package

Add a `Notifier` callback type and field to `LocalProxyManager` / `LocalProxyManagerConfig`.

**File**: `pkg/commands/mcp/proxy/proxy_manager.go`

### 2. Emit notifications in ExecuteDynamicTool

Before `m.approver.RequestApproval(...)`:
- Emit warning: `"Risky operation detected. Waiting for approval in Apono..."`

After approval resolves:
- Approved: emit info `"Operation approved by <responder>"`
- Denied: emit error `"Operation denied by <responder>"`

**File**: `pkg/commands/mcp/proxy/proxy_manager.go`

### 3. Declare logging capability

Add `"logging": {}` to the capabilities map in the MCP initialize response.

**File**: `pkg/commands/mcp/actions/handler.go`

### 4. Wire notifier to stdout in MCP server

Create a notifier function that writes JSON-RPC notifications to stdout. Use a mutex to protect concurrent writes (approval polling runs in the tool-call goroutine while the main loop also writes to stdout).

**File**: `pkg/commands/mcp/actions/mcp.go`
