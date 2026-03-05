# Apono Vault CLI Commands - Design

**Tickets**: DVL-8148 (manage secrets), DVL-8149 (view secret)
**Date**: 2026-03-03
**Author**: Tomer Stein + Claude

## Commands

```
apono vault
  fetch <path> --vault-id <id|name> [--format json|table]     # requires secret grant
  list --vault-id <id|name> [--format json|table]              # requires management grant
  create <path> --vault-id <id|name> --value '<json>'          # requires management grant
  update <path> --vault-id <id|name> --value '<json>'          # requires management grant
  delete <path> --vault-id <id|name>                           # requires management grant
```

### Flags

| Flag | Description |
|------|-------------|
| `--vault-id` | Integration ID (stable) or name (human-friendly). Required on all commands. |
| `--value` | Secret value as JSON string, e.g. `'{"username": "admin", "password": "s3cr3t"}'` |
| `--format` | Output format: `table` (default) or `json` |

## File Structure

```
pkg/commands/vault/
  configurator.go
  actions/
    vault.go        # parent command
    fetch.go
    list.go
    create.go
    update.go
    delete.go

pkg/services/vault.go   # session discovery, credential caching, vault HTTP ops
```

Register `&vault.Configurator{}` in `pkg/commands/apono/runner.go`.
Parent command in `ManagementCommandsGroup`.

## Credential Flow

Every vault command follows this flow:

1. Get Apono client from context.
2. Resolve `--vault-id` to integration (supports both ID and name via existing `GetIntegrationByIDOrByTypeAndName`).
3. List active sessions filtered by integration ID.
   - `fetch`: needs `session-apono-vault-secret` session type.
   - `list/create/update/delete`: needs `session-apono-vault-management` session type.
4. No matching session -> error: "No active vault access for '<vault-id>'. Request access first."
5. Check credential cache at `~/.apono/cache/vault-<integration-id>`:
   - Cache hit -> use cached vault_address, username, password.
   - Cache miss -> call access details API for the session.
     - If credentials available (password present) -> save to cache, use them.
     - If credentials not available -> error: "Run `apono access reset-credentials <session-id>` then retry."
6. Login to vault: `POST <vault_address>/v1/auth/userpass/login/<username>` -> get vault token.
7. Execute vault operation with token.
8. Print output.

### Credential Cache

- Path: `~/.apono/cache/vault-<integration-id>`
- Format: base64-encoded JSON: `{"vault_address": "...", "username": "...", "password": "..."}`
- Keyed by integration ID (stable, survives renames).
- Same credentials for same user within same integration (Apono platform behavior).
- All vault commands for a vault-id share the same cache.
- Re-login to vault on every command (stateless, single HTTP call, acceptable latency).

## Vault HTTP Operations

Plain HTTP calls, no vault SDK dependency.

User-facing paths use short form: `kv/db/production`.
CLI maps to KV v2 API paths by inserting `data/` after mount: `kv/data/db/production`.
Mount name = first segment of the path.

| Command | HTTP Method | Vault API Path | Body |
|---------|-------------|----------------|------|
| `fetch kv/db/prod` | GET | `/v1/kv/data/db/prod` | - |
| `list` | LIST | `/v1/<mount>/metadata/` | - |
| `create kv/new` | POST | `/v1/kv/data/new` | `{"data": <json-value>}` |
| `update kv/existing` | POST | `/v1/kv/data/existing` | `{"data": <json-value>}` |
| `delete kv/old` | DELETE | `/v1/kv/data/old` | - |

### Path Parsing

Given user input `kv/db/prod`:
- Mount: `kv` (first path segment)
- Secret path: `db/prod` (remaining segments)
- API path: `kv/data/db/prod`

For `list`, mount comes from the session's `mount_name` param.

## Output Formatting

- `fetch` table: key/value pairs of the secret data.
- `fetch` json: raw vault API response data.
- `list` table: secret paths, one per row.
- `list` json: array of secret paths.
- `create/update/delete`: success message or error.

## Integration Config Changes Needed

The vault integration config (`apono-vault.json`) needs updates for both session types:

```diff
  "cred_params": [
-   "username"
+   "username",
+   "password"
  ],
  "common_params": [
-   []
+   "session_id"
  ],
```

Add `credentials` template in `access_details_templates`:
```json
{
  "vault_address": "{{{params.vault_address}}}",
  "username": "{{{cred_params.username}}}",
  "password": "{{{cred_params.password}}}"
}
```

## Error Messages

| Condition | Message |
|-----------|---------|
| No active session | "No active vault access for '<vault-id>'. Request access via `apono access` or the Apono portal." |
| No cached creds + creds unavailable | "Credentials not available. Run `apono access reset-credentials <session-id>` then retry." |
| Vault login failed | "Failed to authenticate to vault at <address>: <error>" |
| Vault operation failed | "Vault operation failed: <error>" |
| Invalid --value JSON | "Invalid JSON value: <parse-error>" |
| Secret not found | "Secret '<path>' not found in vault '<vault-id>'." |
