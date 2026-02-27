# Generic MCP Proxy Design

**Date:** 2026-02-27
**Branch:** Plaiyng-with-dual-cred-flow
**Status:** Approved

## Goal

Allow agents to seamlessly get MCP servers with credentials injected from Apono sessions. The system must be generic (support any MCP server via config), start with postgres (@anthropic-ai/postgres-mcp-server), and auto-spawn MCP servers when access is granted — no explicit lifecycle management by the agent.

## Architecture: Smart Gateway

The proxy is a gateway that manages a registry of running backends, handles tool routing, and applies cross-cutting concerns (risk detection, logging, approval).

```
Agent (Claude, Cursor, etc.)
  | JSON-RPC via stdio
  v
MCPHandler (handler.go)
  |-- Static Tools (ToolRegistry)
  |   |-- list_available_resources
  |   |-- ask_access_assistant
  |   |-- create_access_request
  |   |-- get_request_details
  |   +-- list_targets
  |
  +-- LocalProxyManager (proxy_manager.go)
      |-- MCP Server Registry        <-- loads mcp-servers.yaml
      |   +-- maps integration_type -> server definition
      |
      |-- CredentialBuilder          <-- renders Go templates
      |   +-- raw session fields -> composed credentials
      |
      |-- Session Watcher            <-- polls sessions, auto-spawns/kills
      |   |-- detects new sessions -> spawn matching MCP server
      |   +-- detects expired sessions -> kill backend + notify
      |
      +-- Running Backends (map[targetID] -> STDIOBackend)
          +-- e.g. @anthropic-ai/postgres-mcp-server
              |-- query
              |-- list_tables
              +-- describe_table
```

## YAML Config Format

Location: `~/.apono/mcp-servers.yaml`

```yaml
mcp_servers:
  - id: postgres
    name: "PostgreSQL MCP"
    # Apono integration types that trigger this MCP server
    integration_types: ["postgresql", "postgres", "rds-postgresql"]
    # How to spawn the subprocess
    command: "npx"
    args: ["-y", "@anthropic-ai/postgres-mcp-server"]
    # Compose credentials from raw Apono session fields using Go templates
    credential_builder:
      database_url: "postgresql://{{.username}}:{{.password}}@{{.host}}:{{.port}}/{{.db_name}}?sslmode=require"
    # Inject composed credentials as environment variables
    env_mapping:
      database_url: "DATABASE_URL"
    # Inject composed credentials as positional args (appended in order)
    arg_mapping:
      - database_url
```

### Config fields

| Field | Required | Description |
|---|---|---|
| `id` | yes | Unique identifier for this MCP server type |
| `name` | yes | Human-readable name |
| `integration_types` | yes | List of Apono integration type strings that match this server |
| `command` | yes | Executable to run |
| `args` | no | Base arguments (before credential args) |
| `credential_builder` | no | Go templates that compose raw session fields into derived credential keys. If omitted, raw session fields pass through directly. |
| `env_mapping` | no | Map of credential key -> environment variable name |
| `arg_mapping` | no | Ordered list of credential keys to append as positional args |

### Template functions

- Standard Go `text/template` syntax: `{{.field_name}}`
- `{{urlEncode .password}}` — URL-encodes a value (for connection strings with special chars)

## Credential Builder

Replaces the hardcoded per-integration-type switch in `session_provider.go`.

**Flow:**
1. Apono session provides raw fields as `map[string]string`: `{"host": "db.prod.com", "port": "5432", "username": "app_user", "password": "secret", "db_name": "mydb"}`
2. For each key in `credential_builder`, render the Go template against raw fields
3. Output: composed `map[string]string`: `{"database_url": "postgresql://app_user:secret@db.prod.com:5432/mydb?sslmode=require"}`
4. `env_mapping` and `arg_mapping` reference these composed credential keys
5. `STDIOBackend.Start()` receives env vars and args, passes to subprocess

**Error handling:** Missing template field -> clear error returned to agent explaining which field is missing from the session.

**Adding a new integration type is purely config — zero Go code changes:**

```yaml
- id: mysql
  name: "MySQL MCP"
  integration_types: ["mysql", "rds-mysql", "aurora-mysql"]
  command: "npx"
  args: ["-y", "@anthropic-ai/mysql-mcp-server"]
  credential_builder:
    database_url: "mysql://{{.username}}:{{.password}}@{{.host}}:{{.port}}/{{.db_name}}"
  arg_mapping:
    - database_url
```

## Session Watcher

Single component that handles both auto-spawn and auto-kill. Replaces `setup_database` builtin tool and idle cleanup.

**Responsibilities:**
- Polls Apono sessions API periodically (every 10 seconds)
- Detects new active sessions -> matches integration type against `mcp-servers.yaml` -> renders credentials -> spawns STDIOBackend -> sends `tools/list_changed`
- Detects expired/revoked sessions -> kills STDIOBackend -> removes from registry -> sends `tools/list_changed`
- Logs all events via `McpLogf`

**Data structure:**
```go
type TrackedBackend struct {
    TargetID    string
    SessionID   string
    ExpiresAt   *time.Time  // nil for file-based targets
    LastUsed    time.Time
}
```

**Coexistence with idle cleanup:** TTL-based cleanup for session targets, idle-based cleanup for file targets. Both run in the same goroutine.

## Agent-Facing Tools

5 granular tools with flow guidance in both descriptions and responses.

### Tool 1: list_available_resources (existing)

**Purpose:** Show what integrations exist and their access status.

**Description includes:** Full 5-step flow overview.

**Response:**
```json
{
  "resources": [
    {"name": "prod-postgres", "type": "postgresql", "status": "needs_access"},
    {"name": "staging-postgres", "type": "postgresql", "status": "ready"}
  ],
  "next_step": "Use ask_access_assistant with your task description to determine what access to request"
}
```

### Tool 2: ask_access_assistant (existing)

**Purpose:** AI-powered scoping of what to request.

**Response includes:** Recommended scope, bundles, permissions, and hint to call `create_access_request`.

### Tool 3: create_access_request (existing)

**Purpose:** Request access with specific scope.

**Response:**
```json
{
  "request_id": "req-123",
  "status": "pending_approval",
  "next_step": "Use get_request_details to check approval status. Once approved, database tools will automatically become available."
}
```

### Tool 4: get_request_details (existing)

**Purpose:** Check request/approval status.

**Response when granted:**
```json
{
  "request_id": "req-123",
  "status": "approved",
  "session_active": true,
  "next_step": "Access granted. Database tools are now available in your tool list. Use list_targets to see connected targets and their tools."
}
```

### Tool 5: list_targets (existing, enriched)

**Purpose:** Show connected targets with their available tools.

**Response:**
```json
{
  "connected_targets": [
    {
      "name": "prod-postgres",
      "type": "postgresql",
      "status": "connected",
      "session_expires_in": "2h 15m",
      "available_tools": [
        "prod_postgres__query",
        "prod_postgres__list_tables",
        "prod_postgres__describe_table"
      ]
    }
  ],
  "pending_requests": [
    {"name": "staging-mysql", "status": "pending"}
  ],
  "next_step": "You can now use the tools listed above directly."
}
```

## Auto-Spawn Flow (End to End)

1. Agent calls `list_available_resources` -> sees `prod-postgres` with `status: needs_access`
2. Agent calls `ask_access_assistant` with task context -> gets recommended scope
3. Agent calls `create_access_request` with scope -> gets `request_id`, `status: pending`
4. Agent calls `get_request_details` -> polls until `status: approved`
5. **Session Watcher** detects new active session for `postgresql` integration
6. Watcher looks up `mcp-servers.yaml` -> finds postgres MCP server definition
7. Watcher renders `credential_builder` templates with raw session credentials
8. Watcher spawns `STDIOBackend` with composed env vars and args
9. Backend initializes, tools discovered
10. Proxy sends `tools/list_changed` notification
11. Agent sees postgres tools (`query`, `list_tables`, `describe_table`) appear in its tool list
12. Agent uses tools directly — no setup step needed

## Session Expiry Flow

1. Session Watcher detects session TTL expired (or session revoked via API)
2. Watcher kills the `STDIOBackend` subprocess
3. Watcher removes backend from registry
4. Proxy sends `tools/list_changed` notification
5. Agent sees postgres tools disappear
6. If agent tries to call a disappeared tool, it gets: "Session expired for prod-postgres. Use list_available_resources to check status and request new access."

## Changes vs. Current Code

| Component | Current | New |
|---|---|---|
| `BackendTypeConfig` | Hardcoded in Go / partial in targets.yaml | Loaded from `~/.apono/mcp-servers.yaml` |
| Credential composition | Per-type switch in `session_provider.go` | Go template rendering from YAML config |
| Integration -> MCP mapping | Implicit in `setup_database` builtin tool | Declarative via `integration_types` field |
| MCP spawn trigger | Agent calls `init_target` or `setup_database` | Auto-spawn by Session Watcher on access grant |
| Session expiry | Idle timeout cleanup only | Active TTL monitoring + kill + `tools/list_changed` |
| Builtin proxy tools | `list_targets`, `init_target`, `stop_target`, `setup_database` | `list_targets` only (enriched with tool listing) |
| Adding new integration | Go code changes | YAML config only |

## Starting Point: Postgres

First implementation uses:
- MCP server: `@anthropic-ai/postgres-mcp-server`
- Apono integration types: `postgresql`, `postgres`, `rds-postgresql`
- Credential: connection string via `database_url`
- Injection: arg_mapping (connection string as positional arg)

## Key Design Decisions

1. **Smart Gateway over Thin Proxy** — cross-cutting concerns (risk, approval, logging) need a central interception point
2. **Auto-spawn over explicit setup** — agent never manages MCP lifecycle, tools appear/disappear automatically
3. **Go templates over hardcoded switches** — new integrations are config-only, no Go code changes
4. **Local YAML config** — users own the MCP server mappings, can add custom servers
5. **Granular tools with flow hints** — 5 explicit tools with guidance in descriptions and responses, rather than consolidated or intent-based approaches
6. **Session Watcher as single lifecycle component** — handles both spawn (new sessions) and cleanup (expired sessions)
