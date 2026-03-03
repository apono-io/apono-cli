# Apono Claude Code Plugin Design

## Goal

Create a Claude Code plugin that integrates Apono's MCP proxy into Claude Code, allowing agents to securely access databases and services through Apono's access management. Users control which targets the agent can access via an interactive skill.

## Architecture

### Plugin Type

Claude Code plugin, living inside the `apono-cli` repo at `plugin/`.

### How It Works

```
┌─────────────────────────────────────────────────┐
│ Claude Code Session                             │
│                                                 │
│  User runs /apono:connect                       │
│       │                                         │
│       ▼                                         │
│  Skill: list_targets → show targets → user picks│
│       │                                         │
│       ▼                                         │
│  Skill: setup_database for each selected target │
│       │                                         │
│       ▼                                         │
│  MCP Proxy spawns backends → tools available    │
│       │                                         │
│       ▼                                         │
│  Agent uses query/execute tools on databases    │
│                                                 │
│  ┌─────────────────────────────────────┐        │
│  │ apono mcp --proxy (stdio)           │        │
│  │  ├─ list_targets (always available) │        │
│  │  ├─ setup_database (always avail.)  │        │
│  │  └─ query, execute... (after spawn) │        │
│  └─────────────────────────────────────┘        │
└─────────────────────────────────────────────────┘
```

### Lifecycle

1. **Plugin loads** → `.mcp.json` auto-starts `apono mcp --proxy` as a stdio MCP server. This is lightweight — no active backends yet, just the proxy process waiting for requests.
2. **User invokes `/apono:connect`** → Skill fetches available targets, presents them interactively, user selects which targets to expose.
3. **Skill activates targets** → Calls `setup_database` for each selected target. The proxy spawns backend MCP servers with injected credentials.
4. **Agent works** → Dynamic tools (e.g., `query`, `execute`) are now available. Agent uses them to interact with databases.
5. **Session ends** → Proxy shuts down, all backends are cleaned up.

## Plugin Structure

```
plugin/
├── .claude-plugin/
│   └── plugin.json           # Plugin manifest
├── skills/
│   └── connect/
│       ├── SKILL.md           # /apono:connect skill
│       └── reference.md       # Detailed usage guide
├── .mcp.json                  # Auto-starts apono mcp --proxy
└── README.md                  # Installation and usage docs
```

## Components

### 1. Plugin Manifest (`.claude-plugin/plugin.json`)

```json
{
  "name": "apono",
  "description": "Secure database and service access management via Apono",
  "version": "0.1.0",
  "author": {
    "name": "Apono"
  },
  "repository": "https://github.com/apono-io/apono-cli",
  "keywords": ["database", "access", "security", "mcp", "apono"]
}
```

- **name**: `apono` — skills are namespaced as `/apono:connect`
- **version**: `0.1.0` — initial release

### 2. MCP Server Configuration (`.mcp.json`)

```json
{
  "mcpServers": {
    "apono": {
      "command": "apono",
      "args": ["mcp", "--proxy", "--risk-action", "approve"]
    }
  }
}
```

- Starts `apono mcp --proxy` as a stdio MCP server when the plugin loads
- `--risk-action approve` is the default — risky operations require Apono approval
- Users can override this by redefining the `apono` MCP server in their own Claude Code settings

### 3. Connect Skill (`skills/connect/SKILL.md`)

The primary user-facing skill. Handles target discovery and selection.

**Frontmatter:**

```yaml
---
name: connect
description: >
  Connect to databases and services through Apono. Use when the user
  wants to access a database, query data, or set up access to a service.
allowed-tools: Bash(apono *)
---
```

- Both user-invocable (`/apono:connect`) and model-invocable (Claude triggers when user asks about database access)
- `allowed-tools`: permits running `apono` CLI commands without per-use approval

**Skill flow:**

1. Verify `apono` CLI is installed and authenticated
2. Call MCP `list_targets` to discover available targets
3. Present targets to user grouped by type with status indicators
4. User selects targets to activate
5. Call `setup_database` for each "ready" target
6. Confirm active targets and available tools

**Argument handling:**

- No arguments: interactive target selection
- Keyword (e.g., `/apono:connect staging`): filter targets by name
- `--all`: include non-database integrations

### 4. Reference Guide (`skills/connect/reference.md`)

Detailed documentation covering:
- All available MCP tools and their parameters
- Target statuses and what they mean
- Troubleshooting common issues (auth failures, no targets, etc.)
- How to customize `.mcp.json` flags

## Configuration

### Server-level (`.mcp.json`)

Controls how the proxy runs. Key flags:

| Flag | Default | Description |
|------|---------|-------------|
| `--risk-action` | `approve` | `deny` (block risky ops), `approve` (Apono approval flow), `allow` (no checks) |
| `--all-integrations` | off | Show all integration types, not just databases |
| `--targets-file` | `~/.apono/mcp-proxy/targets.yaml` | Static targets file path |
| `--mcp-servers-file` | `~/.apono/mcp-servers.yaml` | MCP server definitions file |

Users override by editing the `.mcp.json` or redefining the server in their Claude Code settings.

### Skill-level (`$ARGUMENTS`)

Session-specific choices passed to `/apono:connect`.

## Prerequisites

- `apono` CLI installed and in PATH
- User authenticated via `apono auth login`
- Apono sessions or integrations configured in Apono platform

## User Experience

### Happy Path

```
User: I need to check something in the staging database

Claude: I'll help you connect. Let me check available targets.
        [calls list_targets]

        Available targets:
        1. staging-postgres (PostgreSQL) - ready
        2. prod-postgres (PostgreSQL) - needs_access
        3. analytics-redshift (Redshift) - ready

        Which targets would you like to activate?

User: Just staging-postgres

Claude: [calls setup_database for staging-postgres]
        Connected to staging-postgres. I now have access to query it.
        What would you like to check?

User: Show me all users created this week

Claude: [calls query tool on staging-postgres]
        Here are the users created this week: ...
```

### Auth Not Configured

```
User: /apono:connect

Claude: I checked and it looks like you're not authenticated with Apono.
        Please run `apono auth login` to authenticate first.
```

## Non-Goals (v1)

- Access control policies within the plugin (deferred)
- Automatic target selection without user input
- Multi-profile support in the skill
- Custom MCP server definitions through the plugin

## Future Considerations

- Add a `/apono:disconnect` skill to tear down specific backends
- Support env var configuration (APONO_RISK_ACTION, etc.)
- Add a SessionStart hook to check auth status on startup
- Policy-based auto-approval for read-only operations
- Marketplace distribution
