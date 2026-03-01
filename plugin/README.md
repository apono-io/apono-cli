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
