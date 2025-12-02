# Apono CLI - Credentials Management Commands

This document describes the credentials management commands for saving and using access session credentials.

## Commands Overview

### 1. `apono access save-creds` - Save credentials as env file

**Usage:**
```bash
apono access save-creds <session_id>
```

**What it does:**
- Fetches credentials from the session (using JSON output format)
- Saves them to `~/.config/apono-cli/<session_id>.creds`
- Formats as sourceable environment variables

**Example:**
```bash
apono access save-creds postgresql-local-postgres-adce68
```

**Example output file** (`~/.config/apono-cli/postgresql-local-postgres-adce68.creds`):
```bash
# Apono credentials for session: postgresql-local-postgres-adce68
# Generated at:
# Source this file to load credentials into your environment:
#   source ~/.config/apono-cli/postgresql-local-postgres-adce68.creds

export DB_NAME="postgres"
export HOST="kubernetes.docker.internal"
export PASSWORD="*****"
export PORT="5432"
export USERNAME="dima_agent_apono"

# PostgreSQL connection string
export DATABASE_URL="postgresql://${USERNAME}:${PASSWORD}@${HOST}:${PORT}/${DB_NAME}"
export PGHOST="${HOST}"
export PGPORT="${PORT}"
export PGUSER="${USERNAME}"
export PGPASSWORD="${PASSWORD}"
export PGDATABASE="${DB_NAME}"
```

**Usage in terminal:**
```bash
# Source the credentials file
source ~/.config/apono-cli/postgresql-local-postgres-adce68.creds

# Now you can use the environment variables
echo $DB_NAME
echo $DATABASE_URL

# Connect to PostgreSQL
psql $DATABASE_URL
# or
psql -h $PGHOST -p $PGPORT -U $PGUSER -d $PGDATABASE
```

---

### 2. `apono access setup-mcp` - Auto-configure MCP for AI Clients

**Usage:**
```bash
apono access setup-mcp <session_id> --client <client_type>
```

**Supported Clients:**
- `claude-desktop` - Claude Desktop app
- `claude-code` - Claude Code CLI
- `cursor` - Cursor IDE

**What it does:**
1. Saves credentials to `.creds` file (same as `save-creds`)
2. Automatically configures the MCP server for the specified client
3. Updates the appropriate config file based on client type

**MCP Server Modes:**
- **Read-write (default)**: Full database access including INSERT/UPDATE/DELETE
- **Read-only (--read-only flag)**: Only SELECT queries allowed

Note: Read-write is the default since Apono provides access control at the credential level.

**Examples:**
```bash
# For Claude Desktop (read-write by default)
apono access setup-mcp postgresql-local-postgres-adce68 --client claude-desktop

# For Cursor IDE (read-write by default)
apono access setup-mcp postgresql-local-postgres-adce68 --client cursor

# For Claude Code CLI (read-write by default)
apono access setup-mcp postgresql-local-postgres-adce68 --client claude-code

# For read-only access (if needed)
apono access setup-mcp postgresql-local-postgres-adce68 --client cursor --read-only
```

**Config File Locations:**

| Client | Config File Path |
|--------|-----------------|
| `claude-desktop` | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| `claude-code` | `~/.config/claude/claude_code_config.json` |
| `cursor` | `~/.cursor/mcp.json` |

**Generated MCP Config Example** (format is same for all clients):
```json
{
  "mcpServers": {
    "postgres-postgresql-local-postgres-adce68": {
      "command": "npx",
      "args": [
        "-y",
        "@modelcontextprotocol/server-postgres",
        "postgresql://dima_agent_apono:*****@kubernetes.docker.internal:5432/postgres"
      ]
    }
  }
}
```

**Output Example (for Cursor):**
```
✓ Credentials saved to: ~/.config/apono-cli/postgresql-local-postgres-adce68.creds
✓ MCP server config added to Cursor: ~/.cursor/mcp.json

MCP Server Name: postgres-postgresql-local-postgres-adce68

📝 Next steps:
1. Restart Cursor IDE to load the new MCP server
2. Open Settings > Developer > Edit Config > MCP Tools to verify
3. The PostgreSQL MCP server will appear in Available Tools

Alternatively, you can source the credentials in your terminal:
  source ~/.config/apono-cli/postgresql-local-postgres-adce68.creds
```

---

## Quick Start Guide

### Option 1: Save credentials for manual terminal use

```bash
# 1. Save credentials
apono access save-creds postgresql-local-postgres-adce68

# 2. Source them in your terminal
source ~/.config/apono-cli/postgresql-local-postgres-adce68.creds

# 3. Use with psql or any PostgreSQL client
psql $DATABASE_URL

# 4. Or use individual env vars
psql -h $PGHOST -p $PGPORT -U $PGUSER -d $PGDATABASE
```

### Option 2: Auto-configure for AI Clients (Claude Desktop, Claude Code, Cursor)

```bash
# 1. Setup MCP server for your preferred client
apono access setup-mcp postgresql-local-postgres-adce68 --client cursor

# 2. Restart the client application (completely quit and reopen)
# The MCP server will now be available!

# 3. You can now ask your AI assistant:
#    "Show me all tables in the database"
#    "What is the schema of the users table?"
#    "Run a query to get all active users"
```

---

## Development Testing

### With local development build:

```bash
cd /Users/dmitryogurko/apono/apono-cli

# Test save-creds command
APONO_USER_ID="your-user-id" go run ./cmd/apono access save-creds postgresql-local-postgres-adce68 --profile local

# Check the generated file
cat ~/.config/apono-cli/postgresql-local-postgres-adce68.creds

# Test MCP setup for Cursor (read-write by default)
APONO_USER_ID="your-user-id" go run ./cmd/apono access setup-mcp postgresql-local-postgres-adce68 --client cursor --profile local

# Check config was updated
cat ~/.cursor/mcp.json | jq

# Or for Claude Desktop (read-write by default)
APONO_USER_ID="your-user-id" go run ./cmd/apono access setup-mcp postgresql-local-postgres-adce68 --client claude-desktop --profile local
cat ~/Library/Application\ Support/Claude/claude_desktop_config.json | jq

# For read-only mode (if needed)
APONO_USER_ID="your-user-id" go run ./cmd/apono access setup-mcp postgresql-local-postgres-adce68 --client cursor --read-only --profile local
```

### With alias (recommended):

Add to your `~/.zshrc`:
```bash
apono-local() {
    (cd /Users/dmitryogurko/apono/apono-cli && go run ./cmd/apono "$@")
}
```

Then use:
```bash
# Save credentials
APONO_USER_ID="your-user-id" apono-local access save-creds postgresql-local-postgres-adce68 --profile local

# Setup MCP for Cursor (read-write by default)
APONO_USER_ID="your-user-id" apono-local access setup-mcp postgresql-local-postgres-adce68 --client cursor --profile local
```

---

## Files Created

The following files are created/modified by these commands:

### Credentials File
- **Location:** `~/.config/apono-cli/<session_id>.creds`
- **Format:** Shell script with `export` statements
- **Permissions:** `0600` (read/write for owner only)
- **Purpose:** Source in terminal to load credentials as environment variables

### MCP Client Config Files
- **Claude Desktop:** `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Claude Code:** `~/.config/claude/claude_code_config.json`
- **Cursor:** `~/.cursor/mcp.json`
- **Format:** JSON
- **Permissions:** `0600` (read/write for owner only)
- **Purpose:** Configure MCP servers for AI clients

---

## Environment Variables Set

When you source the `.creds` file, the following environment variables are set:

### Standard Credentials
- `DB_NAME` - Database name
- `HOST` - Database host
- `PORT` - Database port
- `USERNAME` - Database username
- `PASSWORD` - Database password

### PostgreSQL-specific Variables
- `DATABASE_URL` - Full PostgreSQL connection string
- `PGHOST` - PostgreSQL host (same as `HOST`)
- `PGPORT` - PostgreSQL port (same as `PORT`)
- `PGUSER` - PostgreSQL user (same as `USERNAME`)
- `PGPASSWORD` - PostgreSQL password (same as `PASSWORD`)
- `PGDATABASE` - PostgreSQL database (same as `DB_NAME`)

---

## Requirements

### For `save-creds` command:
- No external dependencies
- Just needs access to the Apono API

### For `setup-mcp` command:
- One of the supported AI clients installed (Claude Desktop, Claude Code, or Cursor)
- Node.js/npm installed (for `npx` to run MCP servers)
- PostgreSQL MCP server will be downloaded automatically on first use via `npx`

---

## MCP Server Details

The MCP (Model Context Protocol) server allows AI clients to directly interact with your database.

**MCP Server Used (by default):** `@henkey/postgres-mcp-server`

**Capabilities (17 consolidated tools):**
- **Execute Query** - SELECT operations with count/exists
- **Execute Mutation** - INSERT/UPDATE/DELETE/UPSERT operations
- **Execute SQL** - Arbitrary SQL with transactions
- Schema Management (tables, columns, constraints)
- User & Permissions management
- Query Performance analysis
- Index Management
- Functions, Triggers, Row-Level Security
- Database Analysis and Monitoring
- Data Export/Import

**Read-only mode:** Use `--read-only` flag to use `@modelcontextprotocol/server-postgres` for SELECT-only access.

**Security Note:**
- Credentials are stored in plain text in the client's config file
- The config file has `0600` permissions (owner-only access)
- Read-write access is enabled by default (Apono controls access at the credential level)
- Use `--read-only` flag if you only need SELECT queries
- Apono manages access control, permissions, and audit logging

---

## Troubleshooting

### Credentials file not found
```bash
# Check if the config directory exists
ls -la ~/.config/apono-cli/

# If it doesn't exist, create it
mkdir -p ~/.config/apono-cli
```

### AI Client not loading MCP server

**For Claude Desktop:**
1. Completely quit Claude Desktop (not just close the window)
2. Check the config file is valid JSON:
   ```bash
   cat ~/Library/Application\ Support/Claude/claude_desktop_config.json | jq
   ```
3. Check Claude Desktop logs:
   ```bash
   tail -f ~/Library/Logs/Claude/mcp*.log
   ```

**For Cursor:**
1. Completely quit Cursor IDE and restart
2. Check the config file is valid JSON:
   ```bash
   cat ~/.cursor/mcp.json | jq
   ```
3. Open Settings > Developer > Edit Config > MCP Tools to verify server appears
4. Check Cursor logs for MCP errors

**For Claude Code:**
1. Check the config file is valid JSON:
   ```bash
   cat ~/.config/claude/claude_code_config.json | jq
   ```
2. Use `claude mcp list` to verify the server is loaded
3. Reload MCP configuration if needed

### MCP server fails to connect
1. Verify credentials work manually:
   ```bash
   source ~/.config/apono-cli/postgresql-local-postgres-adce68.creds
   psql $DATABASE_URL
   ```
2. Check if `npx` is available:
   ```bash
   npx --version
   ```
3. Manually test the MCP server:
   ```bash
   npx -y @modelcontextprotocol/server-postgres "$DATABASE_URL"
   ```

---

## Example Workflows

### Workflow 1: Quick database access in terminal
```bash
# Get credentials and start working immediately
apono access save-creds my-db-session
source ~/.config/apono-cli/my-db-session.creds
psql $DATABASE_URL
```

### Workflow 2: AI assistant for database queries
```bash
# Setup once for your preferred client
apono access setup-mcp my-db-session --client cursor

# Restart the client
# Now ask your AI assistant to help with database queries!
```

### Workflow 3: Scripting with credentials
```bash
# Save credentials
apono access save-creds my-db-session

# Use in a script
source ~/.config/apono-cli/my-db-session.creds
python my_script.py  # Script can use $PGHOST, $PGPASSWORD, etc.
```

---

## Related Commands

- `apono access list` - List all available access sessions
- `apono access use <session_id>` - Get session credentials (various formats)
- `apono access reset-credentials <session_id>` - Reset session credentials

---

## Security Best Practices

1. **Never commit `.creds` files to version control**
   - Add to `.gitignore`: `*.creds`

2. **Credentials have limited lifetime**
   - Apono sessions expire based on policy
   - Re-run `save-creds` or `setup-mcp` when credentials are rotated

3. **Use read-only access when possible**
   - Request read-only database permissions in Apono
   - Limits blast radius if credentials are compromised

4. **Clean up old credential files**
   ```bash
   # Remove old credential files
   rm ~/.config/apono-cli/*.creds
   ```

5. **Rotate credentials regularly**
   ```bash
   # Reset and get new credentials
   apono access reset-credentials <session_id>
   apono access save-creds <session_id>
   ```
