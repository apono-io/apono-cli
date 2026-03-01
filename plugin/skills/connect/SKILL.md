---
name: connect
description: >
  Connect to databases and services through Apono. Use when the user
  wants to access a database, query data, or set up access to a service
  managed by Apono.
allowed-tools: Bash(apono *)
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
| `list_resources_filtered` | List resources filtered by integration type or keyword |

After a target is activated, additional tools become available from the backend (e.g., `query` for PostgreSQL).

## Troubleshooting

- **No targets visible**: User may not have any active Apono sessions. Suggest using `list_available_resources` to see integrations and `create_access_request` to request access.
- **Auth error**: User needs to run `apono auth login`.
- **Target stuck in pending**: Access request is awaiting approval in Apono.
