# Apono Claude Code Plugin Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a Claude Code plugin that integrates Apono's MCP proxy, giving agents secure access to databases and services via an interactive `/apono:connect` skill.

**Architecture:** Plugin lives at `plugin/` in repo root. `.mcp.json` auto-starts `apono mcp --proxy` as a stdio MCP server. A `/apono:connect` skill orchestrates target discovery and selection. The proxy's session watcher auto-spawns backends with credentials for selected targets.

**Tech Stack:** Claude Code plugin system (plugin.json manifest, SKILL.md files, .mcp.json MCP config). No Go code changes needed — this is purely plugin authoring.

**Design doc:** `docs/plans/2026-03-01-apono-claude-code-plugin-design.md`

---

### Task 1: Create Plugin Manifest

**Files:**
- Create: `plugin/.claude-plugin/plugin.json`

**Step 1: Create the plugin directory structure**

Run: `mkdir -p plugin/.claude-plugin`

**Step 2: Write the plugin manifest**

Create `plugin/.claude-plugin/plugin.json`:

```json
{
  "name": "apono",
  "description": "Secure database and service access management via Apono. Connects AI agents to databases through Apono's access control.",
  "version": "0.1.0",
  "author": {
    "name": "Apono",
    "url": "https://www.apono.io"
  },
  "repository": "https://github.com/apono-io/apono-cli",
  "keywords": ["database", "access", "security", "mcp", "apono"]
}
```

**Step 3: Verify file exists and is valid JSON**

Run: `cat plugin/.claude-plugin/plugin.json | python3 -m json.tool`
Expected: Pretty-printed JSON output without errors.

**Step 4: Commit**

```bash
git add plugin/.claude-plugin/plugin.json
git commit -m "feat(plugin): add Claude Code plugin manifest"
```

---

### Task 2: Create MCP Server Configuration

**Files:**
- Create: `plugin/.mcp.json`

**Step 1: Write the MCP server config**

Create `plugin/.mcp.json`:

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

This tells Claude Code to auto-start `apono mcp --proxy` as a stdio MCP server when the plugin loads. The `--risk-action approve` flag means risky operations (DROP, DELETE, etc.) will go through Apono's approval flow by default.

**Step 2: Verify file is valid JSON**

Run: `cat plugin/.mcp.json | python3 -m json.tool`
Expected: Pretty-printed JSON output without errors.

**Step 3: Commit**

```bash
git add plugin/.mcp.json
git commit -m "feat(plugin): add MCP server configuration for apono proxy"
```

---

### Task 3: Create the Connect Skill

**Files:**
- Create: `plugin/skills/connect/SKILL.md`

**Step 1: Create the skill directory**

Run: `mkdir -p plugin/skills/connect`

**Step 2: Write the SKILL.md**

Create `plugin/skills/connect/SKILL.md`:

```markdown
---
name: connect
description: >
  Connect to databases and services through Apono. Use when the user
  wants to access a database, query data, or set up access to a service
  managed by Apono.
---

# Apono Connect

Help the user connect to databases and services through Apono's MCP proxy.

## Prerequisites

Before proceeding, verify:

1. **Check CLI is installed**: Run `apono --version`. If not found, tell the user to install it: `brew install apono-io/tap/apono-cli` (macOS) or see https://docs.apono.io/docs/cli-installation
2. **Check authentication**: Run `apono auth status`. If not authenticated, tell the user to run `apono auth login` first and stop.

## Connect Flow

### Step 1: Discover Available Targets

Call the MCP tool `list_targets` (no arguments needed).

This returns:
- **connected_targets**: Targets with running backends and available tools
- **pending_requests**: Targets where access has been requested but not yet granted

### Step 2: Present Targets to the User

Show the targets grouped by status:

**Connected (ready to use):**
- List each target with its name, type, and available tools

**Pending access:**
- List targets awaiting approval

If no targets are available, tell the user:
- They may need to request access through the Apono platform
- Or use `list_available_resources` to see what integrations exist

### Step 3: Activate Targets

For targets the user wants to connect to:

1. If the target is already connected, confirm it's ready and list available tools
2. If the target needs setup, call `setup_database` with `{"session_id": "<target_session_id>"}`
3. After setup, call `list_targets` again to confirm the backend spawned and show available tools

### Step 4: Confirm

Tell the user:
- Which targets are now active
- What tools are available (e.g., query, execute)
- They can now ask you to run queries or interact with the database

## Argument Handling

- **No arguments**: Show all available targets, let the user choose
- **Keyword** (e.g., `/apono:connect staging`): Filter targets by name containing the keyword
- **`--all`**: Include non-database integration types

## Available MCP Tools Reference

These tools are provided by the Apono MCP proxy:

| Tool | Description |
|------|-------------|
| `list_targets` | List all available targets with status and tools |
| `setup_database` | Activate a target from an Apono session (params: `session_id`) |
| `list_available_resources` | See what integrations exist and their access status |
| `ask_access_assistant` | Describe your task to get scoped access recommendations |
| `create_access_request` | Request access with specific scope |
| `get_request_details` | Check if an access request was approved |

After a target is activated, additional tools become available from the backend (e.g., `query` for PostgreSQL).

## Troubleshooting

- **No targets visible**: User may not have any active Apono sessions. Suggest using `list_available_resources` to see integrations and `create_access_request` to request access.
- **Auth error**: User needs to run `apono auth login`.
- **Target stuck in pending**: Access request is awaiting approval in Apono.
```

**Step 3: Verify the file was created**

Run: `head -5 plugin/skills/connect/SKILL.md`
Expected: Shows the YAML frontmatter starting with `---`.

**Step 4: Commit**

```bash
git add plugin/skills/connect/SKILL.md
git commit -m "feat(plugin): add /apono:connect skill for target selection"
```

---

### Task 4: Create the Reference Guide

**Files:**
- Create: `plugin/skills/connect/reference.md`

**Step 1: Write the reference guide**

Create `plugin/skills/connect/reference.md`:

```markdown
# Apono Connect - Reference Guide

## MCP Server Configuration

The plugin's `.mcp.json` starts `apono mcp --proxy` with these defaults:

```
apono mcp --proxy --risk-action approve
```

### Configurable Flags

Users can customize the MCP server by overriding the `apono` server in their Claude Code settings:

| Flag | Default | Options | Description |
|------|---------|---------|-------------|
| `--risk-action` | `approve` | `deny`, `approve`, `allow` | How to handle risky operations (DROP, DELETE, etc.) |
| `--all-integrations` | off | flag | Include non-database integrations |
| `--targets-file` | `~/.apono/mcp-proxy/targets.yaml` | path | Static targets file |
| `--mcp-servers-file` | `~/.apono/mcp-servers.yaml` | path | MCP server definitions |
| `--debug` | off | flag | Verbose request/response logging |

### Override Example

To change risk action to "allow" (no risk checks), add to your Claude Code MCP settings:

```json
{
  "mcpServers": {
    "apono": {
      "command": "apono",
      "args": ["mcp", "--proxy", "--risk-action", "allow", "--all-integrations"]
    }
  }
}
```

## Target Statuses

| Status | Meaning | Action |
|--------|---------|--------|
| `ready` | Active session with credentials | Can initialize backend immediately |
| `needs_access` | Integration exists, no session | Request access through Apono |
| `pending` | Access request submitted | Wait for approval |
| `connected` | Backend running with tools | Ready to use |

## Workflow

```
list_available_resources  →  See what's available
ask_access_assistant      →  Get access recommendations
create_access_request     →  Request specific access
get_request_details       →  Check approval status
list_targets              →  See connected targets + tools
setup_database            →  Activate a ready target
[dynamic tools]           →  query, execute, etc.
```

## Troubleshooting

### "apono: command not found"

Install the Apono CLI:
- macOS: `brew install apono-io/tap/apono-cli`
- Other: See https://docs.apono.io/docs/cli-installation

### "Authentication failed"

Run `apono auth login` to authenticate.

### No targets after connecting

1. Check if you have active Apono sessions: `apono access list`
2. Request access through Apono platform or use `create_access_request` tool
3. Verify the MCP servers config supports your database type: check `~/.apono/mcp-servers.yaml`

### Backend fails to spawn

Check the MCP log file for errors. The log file location is printed at startup.
Common causes:
- Missing npm packages (e.g., `@anthropic-ai/postgres-mcp-server`)
- Invalid credentials in the Apono session
- Network connectivity to the database
```

**Step 2: Commit**

```bash
git add plugin/skills/connect/reference.md
git commit -m "feat(plugin): add connect skill reference guide"
```

---

### Task 5: Create Plugin README

**Files:**
- Create: `plugin/README.md`

**Step 1: Write the README**

Create `plugin/README.md`:

```markdown
# Apono Plugin for Claude Code

Connect AI agents to databases and services through [Apono](https://www.apono.io) access management.

## What it does

This plugin integrates Apono's MCP proxy with Claude Code, allowing your AI agent to:

- Discover available database targets from your Apono sessions
- Interactively select which databases to connect to
- Run queries and manage data through MCP tools
- Enforce risk detection and approval for dangerous operations

## Prerequisites

1. **Install Apono CLI**: `brew install apono-io/tap/apono-cli`
2. **Authenticate**: `apono auth login`

## Installation

### Local development

```bash
claude --plugin-dir ./plugin
```

### From source

```bash
claude plugin install apono --plugin-dir /path/to/apono-cli/plugin
```

## Usage

Once the plugin is loaded, use the `/apono:connect` skill:

```
/apono:connect              # Interactive target selection
/apono:connect staging      # Filter targets by keyword
```

Or just ask Claude naturally:

```
"I need to query the staging database"
"Connect me to the production PostgreSQL"
```

## Configuration

The MCP proxy starts with these defaults:

- **Risk action**: `approve` (risky operations require Apono approval)
- **Integrations**: databases only

To customize, override the `apono` MCP server in your Claude Code settings. See the [reference guide](skills/connect/reference.md) for details.

## How it works

1. Plugin loads → MCP proxy starts (lightweight, no active backends)
2. You invoke `/apono:connect` → discover and select targets
3. Proxy spawns backend MCP servers with injected credentials
4. Agent uses database tools (query, execute, etc.)
5. Session ends → backends cleaned up
```

**Step 2: Commit**

```bash
git add plugin/README.md
git commit -m "feat(plugin): add README with installation and usage docs"
```

---

### Task 6: Test the Plugin Locally

**Step 1: Verify the complete plugin structure**

Run: `find plugin -type f | sort`

Expected output:
```
plugin/.claude-plugin/plugin.json
plugin/.mcp.json
plugin/README.md
plugin/skills/connect/SKILL.md
plugin/skills/connect/reference.md
```

**Step 2: Validate all JSON files**

Run: `for f in plugin/.claude-plugin/plugin.json plugin/.mcp.json; do echo "=== $f ===" && python3 -m json.tool "$f" > /dev/null && echo "OK" || echo "FAIL"; done`

Expected: Both files show "OK".

**Step 3: Validate SKILL.md frontmatter**

Run: `head -6 plugin/skills/connect/SKILL.md`

Expected: Shows valid YAML frontmatter with `name: connect` and `description:`.

**Step 4: Test with Claude Code (manual)**

Run Claude Code with the plugin loaded:

```bash
claude --plugin-dir ./plugin
```

Then verify:
1. `/help` shows `/apono:connect` in the skill list
2. The Apono MCP server starts (check for `apono` in MCP server list)
3. `/apono:connect` invokes the skill

**Step 5: Commit final state**

If any fixes were needed during testing:
```bash
git add -A plugin/
git commit -m "fix(plugin): fixes from local testing"
```

---

### Task 7: Final Review and Cleanup

**Step 1: Review all files for consistency**

Check that:
- Plugin name is `apono` everywhere
- Skill name is `connect` everywhere
- MCP server command matches the actual CLI (`apono mcp --proxy`)
- Tool names match the actual implementations (`list_targets`, `setup_database`, `list_available_resources`, `ask_access_assistant`, `create_access_request`, `get_request_details`)

**Step 2: Verify no sensitive information in any file**

Run: `grep -ri "password\|secret\|token\|api_key" plugin/`

Expected: No matches (or only in documentation examples that use placeholder values).

**Step 3: Final commit if needed**

```bash
git add plugin/
git commit -m "feat(plugin): Apono Claude Code plugin v0.1.0"
```
