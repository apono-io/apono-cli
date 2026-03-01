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
