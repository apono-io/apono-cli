# Approval Notifications Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Show users real-time MCP logging notifications when a risky operation triggers the approval flow — "Waiting for approval..." on start, "Approved/Denied by X" on result.

**Architecture:** Add a `Notifier` callback (`func(level, message string)`) to the proxy manager. The MCP server wires it to write `notifications/message` JSON-RPC notifications to stdout. The proxy manager calls it from `ExecuteDynamicTool` before and after the approval poll. Uses the same stdout-writing pattern as the existing `toolsChangedFn` callback.

**Tech Stack:** Go, MCP protocol (JSON-RPC 2.0), existing proxy/approval packages.

---

### Task 1: Add Notifier to ProxyManager

**Files:**
- Modify: `pkg/commands/mcp/proxy/proxy_manager.go:57-74` (LocalProxyManager struct)
- Modify: `pkg/commands/mcp/proxy/proxy_manager.go:76-87` (LocalProxyManagerConfig struct)
- Modify: `pkg/commands/mcp/proxy/proxy_manager.go:90-129` (NewLocalProxyManager)

**Step 1: Add Notifier type and field to LocalProxyManager**

In `proxy_manager.go`, add a `Notifier` type alias and a field to the struct:

```go
// Notifier sends MCP logging notifications to the client.
type Notifier func(level, message string)
```

Add to `LocalProxyManager` struct (after `toolsChangedFn`):

```go
notifier Notifier // called to send MCP log notifications to the client
```

Add to `LocalProxyManagerConfig`:

```go
Notifier Notifier // Optional: sends MCP log notifications to the client
```

**Step 2: Wire notifier in NewLocalProxyManager**

In `NewLocalProxyManager`, after setting `done`:

```go
notifier: cfg.Notifier,
```

**Step 3: Add notify helper**

Add a helper method (after `notifyToolsChanged`):

```go
func (m *LocalProxyManager) notify(level, message string) {
	if m.notifier != nil {
		m.notifier(level, message)
	}
}
```

**Step 4: Run tests**

Run: `go build ./pkg/commands/mcp/...`
Expected: compiles with no errors (no test changes needed, this is additive)

**Step 5: Commit**

```
feat(mcp): add Notifier callback to proxy manager
```

---

### Task 2: Emit notifications in ExecuteDynamicTool

**Files:**
- Modify: `pkg/commands/mcp/proxy/proxy_manager.go:232-282` (risk detection + approval block)

**Step 1: Add notification before RequestApproval**

In `ExecuteDynamicTool`, right after the `utils.McpLogf("[ProxyManager] Risk detected for %s:..."` line (line 236) and before the `if m.approver != nil` check, add:

```go
m.notify("warning", fmt.Sprintf("Risky operation detected: %s. Waiting for approval in Apono...", riskResult.Reason))
```

**Step 2: Add notification after approval resolves**

After the `utils.McpLogf("[ProxyManager] Approval result for %s:..."` line (line 261), before the `if !approvalResult.Approved` check, add:

```go
// Notify user of approval result
responder := ""
if r, ok := m.approver.(*approval.ApprovalCache); ok {
	_ = r // cache auto-approvals don't have a responder
}
```

Actually, simpler: the `approvalResult` doesn't carry the responder name (only the `ActionApprovalResponse` does, deep in the approver). Instead, use a generic message:

After the existing `McpLogf` on line 261:

```go
if approvalResult.Approved {
	m.notify("info", "Operation approved. Proceeding with execution.")
} else {
	m.notify("error", fmt.Sprintf("Operation denied: %s", riskResult.Reason))
}
```

**Step 3: Run tests**

Run: `go build ./pkg/commands/mcp/...`
Expected: compiles with no errors

**Step 4: Commit**

```
feat(mcp): emit approval notifications from proxy manager
```

---

### Task 3: Declare logging capability in MCP initialize

**Files:**
- Modify: `pkg/commands/mcp/actions/handler.go:114-130` (handleInitialize)

**Step 1: Add logging capability**

In `handleInitialize`, add `"logging"` to the capabilities map. Change the return from:

```go
return map[string]interface{}{
	"protocolVersion": MCPVersion,
	"capabilities": map[string]interface{}{
		"tools": toolsCap,
	},
```

to:

```go
return map[string]interface{}{
	"protocolVersion": MCPVersion,
	"capabilities": map[string]interface{}{
		"tools":   toolsCap,
		"logging": map[string]interface{}{},
	},
```

**Step 2: Run tests**

Run: `go build ./pkg/commands/mcp/...`
Expected: compiles with no errors

**Step 3: Commit**

```
feat(mcp): declare logging capability in initialize response
```

---

### Task 4: Wire notifier to stdout in MCP server

**Files:**
- Modify: `pkg/commands/mcp/actions/mcp.go:236-261` (runLocalSTDIOServerWithProxy, proxy manager creation + callback wiring)

**Step 1: Create a shared stdout writer with mutex**

Before the proxy manager creation (around line 236), add a mutex-protected writer:

```go
// Mutex-protected stdout writer for concurrent notification writes.
// Both the main loop (responses) and background goroutines (tools_changed, approval notifications)
// write to stdout — this prevents interleaved output.
var stdoutMu sync.Mutex
writeStdout := func(line string) {
	stdoutMu.Lock()
	defer stdoutMu.Unlock()
	fmt.Println(line)
}
```

Add `"sync"` to the imports if not already present.

**Step 2: Create notifier function and pass to config**

Add a `Notifier` to the proxy manager config:

```go
pm := proxy.NewLocalProxyManager(proxy.LocalProxyManagerConfig{
	MCPRegistry:     mcpReg,
	TargetSource:    compositeSource,
	RiskDetector:    riskDetector,
	Approver:        approver,
	APIBaseURL:      apiBaseURL,
	HTTPClient:      apiCfg.HTTPClient,
	TargetsFilePath: targetsFilePath,
	Notifier: func(level, message string) {
		notification, _ := json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "notifications/message",
			"params": map[string]interface{}{
				"level": level,
				"data":  message,
			},
		})
		writeStdout(string(notification))
		utils.McpLogf("Sent notifications/message: level=%s msg=%s", level, message)
	},
})
```

Add `"encoding/json"` to the imports if not already present (it already is).

**Step 3: Update toolsChangedCallback and main loop to use writeStdout**

Update the tools changed callback to use the shared writer:

```go
pm.SetToolsChangedCallback(func() {
	notification := `{"jsonrpc":"2.0","method":"notifications/tools/list_changed"}`
	writeStdout(notification)
	utils.McpLogf("Sent notifications/tools/list_changed")
})
```

Update the main response writer (around line 323-325) to also use writeStdout:

```go
if response != "" {
	utils.McpLogf("[STDIO] << %s (response len=%d)", peek.Method, len(response))
	writeStdout(response)
}
```

**Step 4: Run build**

Run: `go build ./pkg/commands/mcp/...`
Expected: compiles with no errors

**Step 5: Run full test suite**

Run: `go test ./pkg/commands/mcp/...`
Expected: all tests pass

**Step 6: Commit**

```
feat(mcp): wire approval notifications to stdout via MCP logging
```

---

### Task 5: Manual integration test

**Step 1: Build the binary**

Run: `goreleaser build --clean --single-target --snapshot`

**Step 2: Test with MCP test command**

If the `mcp test` subcommand supports sending a risky tool call, use that. Otherwise, verify the build succeeds and the notification JSON format is correct by reviewing the code.

**Step 3: Verify notification format**

The expected notification on stdout when approval is requested:

```json
{"jsonrpc":"2.0","method":"notifications/message","params":{"data":"Risky operation detected: query contains DROP statement. Waiting for approval in Apono...","level":"warning"}}
```

When approved:

```json
{"jsonrpc":"2.0","method":"notifications/message","params":{"data":"Operation approved. Proceeding with execution.","level":"info"}}
```

**Step 4: Commit (if any fixes needed)**

```
fix(mcp): fix approval notification issues from integration test
```
