# Apono MCP Server Setup Guide

This guide explains how to set up and use the Apono MCP (Model Context Protocol) server with Cursor IDE to enable AI-assisted database access management.

## Overview

The Apono MCP server provides tools that allow AI assistants (like Cursor) to:
1. **List active database sessions** - See what databases you currently have access to
2. **List available integrations** - Discover what databases are available through Apono
3. **Setup database MCP servers** - Dynamically configure database connections in Cursor
4. **Request access** - Get information about available access bundles

## Setup

### 1. Configure Cursor to use Apono MCP

Add the Apono MCP server to your Cursor configuration at `~/.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "apono": {
      "command": "/Users/dmitryogurko/apono/apono-cli/apono",
      "args": ["mcp"]
    }
  }
}
```

**Important:** Replace `/Users/dmitryogurko/apono/apono-cli/apono` with the actual path to your built `apono` binary.

Alternatively, if you have `apono` in your PATH:

```json
{
  "mcpServers": {
    "apono": {
      "command": "apono",
      "args": ["mcp"]
    }
  }
}
```

### 2. Ensure you're logged in to Apono

```bash
apono login
```

### 3. Restart Cursor

Completely quit and restart Cursor IDE. The Apono MCP server will be automatically loaded.

## Available Tools

The Apono MCP server exposes three tools designed to support natural, task-oriented conversations:

### 1. `list_available_resources`

Lists all available resources (databases, k8s clusters, etc.) with their connection status.

**When to use:** This should be your **first call** when the user asks about data or infrastructure.

**Parameters:** None

**Returns:**
```json
{
  "resources": [
    {
      "integration_id": "abc123",
      "integration_name": "prod-postgresql",
      "type": "postgresql",
      "type_display_name": "PostgreSQL",
      "status": "active",           // or "available"
      "session_id": "session-xyz",  // present if status="active"
      "session_name": "My DB Session",
      "can_connect": true           // false if needs access request
    }
  ],
  "total": 5,
  "active_count": 2,
  "available_count": 3
}
```

**AI Decision Tree:**
- `status="active"` + `can_connect=true` → Use `setup_database_mcp` with the `session_id`
- `status="available"` + `can_connect=false` → Use `request_access` to show how to get access

### 2. `setup_database_mcp`

Automatically configures a PostgreSQL MCP server in Cursor for an active session.

**Parameters:**
- `session_id` (required): The ID of an active access session

**What it does:**
1. Fetches credentials for the session
2. Creates a PostgreSQL connection string
3. Updates `~/.cursor/mcp.json` with a new PostgreSQL MCP server
4. Cursor automatically reloads the MCP (no restart needed!)

**Returns:**
- Success status, MCP server name, database info

### 3. `request_access`

Get information about access bundles when a resource is not yet accessible.

**Parameters:**
- `integration_id` (required): The integration ID from `list_available_resources`

**Returns:**
- List of available access bundles
- CLI commands for requesting access

## Example Workflows

### Scenario 1: User asks about data (has access)

**User:** "What's the average number of users in our system?"

**AI thinking:** *User is asking about data. I need to check what databases are available.*

**AI (calls `list_available_resources`):**
```json
{
  "resources": [
    {
      "integration_name": "prod-postgresql",
      "type": "postgresql",
      "status": "active",
      "session_id": "session-abc123",
      "can_connect": true
    },
    {
      "integration_name": "staging-postgresql",
      "status": "available",
      "can_connect": false
    }
  ]
}
```

**AI:** "I can see you have access to prod-postgresql (active). Should I check there?"

**User:** "Yes, prod"

**AI (calls `setup_database_mcp` with session_id="session-abc123"):**
```
Success! PostgreSQL MCP connected.
```

**AI (calls PostgreSQL MCP's `Execute Query`):**
```sql
SELECT COUNT(*) as user_count FROM users;
```

**AI:** "There are 45,234 users in the system."

### Scenario 2: User asks about data (no access)

**User:** "Check the staging database performance"

**AI (calls `list_available_resources`):**
```json
{
  "resources": [
    {
      "integration_name": "staging-postgresql",
      "integration_id": "integration-456",
      "status": "available",
      "can_connect": false
    }
  ]
}
```

**AI:** "You don't have active access to staging-postgresql yet."

**AI (calls `request_access` with integration_id="integration-456"):**
```json
{
  "available_bundles": [
    {"name": "Read-only Access"},
    {"name": "DBA Access"}
  ],
  "message": "To request access, run: apono requests new --integration-id integration-456"
}
```

**AI:** "I found 2 access options: Read-only and DBA. To get access, run:\n```\napono requests new --integration-id integration-456\n```\nOnce approved, I'll be able to query the database for you."

### Scenario 3: Deploy to Kubernetes

**User:** "Let's deploy the new service to dev cluster"

**AI (calls `list_available_resources`):**
```json
{
  "resources": [
    {
      "integration_name": "dev-k8s-cluster",
      "type": "kubernetes",
      "status": "active",
      "session_id": "k8s-session-789",
      "can_connect": true
    }
  ]
}
```

**AI:** "You have active access to dev-k8s-cluster. Setting up connection..."

**AI (calls `setup_database_mcp`):** *(Note: Currently only supports PostgreSQL, K8s support would need to be added)*

**AI:** "I see you have access to the dev cluster, but I don't have K8s deployment tools yet. You can deploy using kubectl with your active session credentials."

## AI Decision Flow

This is how the AI should think when processing user requests:

```
User asks: "Show me user statistics"
    ↓
AI calls: list_available_resources
    ↓
Check resources:
    ├─ Found status="active" + can_connect=true?
    │      ↓ YES
    │      Ask user to confirm which resource
    │      ↓
    │      AI calls: setup_database_mcp with session_id
    │      ↓
    │      Use database MCP tools to query
    │
    └─ Found status="available" + can_connect=false?
           ↓ YES
           AI calls: request_access with integration_id
           ↓
           Show user how to request access
           ↓
           Wait for approval, then retry
```

## MCP Servers in Cursor

After `setup_database_mcp` is called, you'll have two MCP servers running:

### 1. Apono MCP (Always available)
Manages database access and permissions:
- `list_available_resources` - See all databases and their status
- `setup_database_mcp` - Connect to an active database
- `request_access` - Get info about requesting access

### 2. PostgreSQL MCP (Dynamically added per database)
Interacts with the actual database:
- `Execute Query` - Run SELECT queries
- `Execute Mutation` - Run INSERT/UPDATE/DELETE
- `Execute SQL` - Run arbitrary SQL with transactions
- Schema management (view tables, columns, constraints)
- Performance analysis (EXPLAIN plans, slow queries)
- Index management

## Configuration Files

### Apono Config
- Profile/credentials: `~/.apono/config.json`
- Session credentials (if saved): `~/.apono/<session-id>.creds`

### Cursor MCP Config
- Location: `~/.cursor/mcp.json`
- Updated automatically by `setup_database_mcp` tool
- Cursor hot-reloads this file (no restart needed)

## Troubleshooting

### "Authentication failed" error

Make sure you're logged in:
```bash
apono login
```

### "MCP server not found" in Cursor

1. Check `~/.cursor/mcp.json` has the apono MCP configured
2. Verify the path to the `apono` binary is correct
3. Restart Cursor completely

### PostgreSQL MCP not working after setup

1. Check `~/.cursor/mcp.json` to see if the postgres MCP was added
2. Look at Cursor's MCP logs for errors
3. The MCP should auto-reload, but try restarting Cursor if needed

### List shows no integrations or sessions

1. Verify your Apono account has access to integrations
2. Check you're using the correct profile: `apono --profile <name> mcp`
3. Try listing via CLI to confirm: `apono access list`

## Advanced Usage

### Using with multiple profiles

You can specify a profile when starting the MCP server:

```json
{
  "mcpServers": {
    "apono-prod": {
      "command": "apono",
      "args": ["mcp", "--profile", "production"]
    },
    "apono-dev": {
      "command": "apono",
      "args": ["mcp", "--profile", "development"]
    }
  }
}
```

### Debug logging

Enable debug logging to see detailed request/response information:

```json
{
  "mcpServers": {
    "apono": {
      "command": "apono",
      "args": ["mcp", "--debug"]
    }
  }
}
```

Logs are written to `~/.apono/mcp.log`

## Security Notes

- The Apono MCP server uses your stored Apono credentials
- Database credentials are dynamically added to `~/.cursor/mcp.json`
- Credentials are only valid for the duration of your Apono session
- Access is controlled by Apono policies and approvals
- All database operations are logged by Apono

## Next Steps

Once set up, try asking Cursor:
- "Show me what databases I have access to"
- "Connect me to the production database"
- "List all tables in my database"
- "What databases are available through Apono?"
