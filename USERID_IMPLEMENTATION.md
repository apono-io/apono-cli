# User ID Implementation Summary

This document describes the implementation of automatic `user_id` query parameter injection for all Access Session API calls.

## Overview

All GET and POST requests to the `/api/client/v1/access-sessions/*` endpoints now automatically include a `user_id` query parameter when available.

## User ID Priority

The `user_id` is sourced in the following priority order:

1. **`APONO_USER_ID` environment variable** (highest priority)
2. **Profile config** - Stored in `~/.config/apono-cli/config.json` from login
3. **No userId** - Falls back to standard API call without the parameter

## Modified Files

### Core Implementation

#### `pkg/aponoapi/access_sessions.go` (NEW FILE)
Contains wrapper methods that add `user_id` query parameter to all access session API calls:

- `GetAccessSessionAccessDetailsWithUserID()` - GET `/api/client/v1/access-sessions/{id}/access-details?user_id=...`
- `ListAccessSessionsWithUserID()` - GET `/api/client/v1/access-sessions?user_id=...&skip=...`
- `GetAccessSessionWithUserID()` - GET `/api/client/v1/access-sessions/{id}?user_id=...`
- `ResetAccessSessionCredentialsWithUserID()` - POST `/api/client/v1/access-sessions/{id}/reset-credentials?user_id=...`

Each method:
1. Checks for `APONO_USER_ID` env var
2. Falls back to `client.Session.UserID` from profile
3. If no userId available, uses standard generated API client
4. If userId available, builds URL manually with query parameter
5. Executes HTTP request with proper auth headers

### Service Layer

#### `pkg/services/sessions.go`
Updated all service methods to use new wrapper methods:

- `ListAccessSessions()` - Line 60: Now uses `ListAccessSessionsWithUserID()`
- `ExecuteAccessDetails()` - Line 78: Now uses `GetAccessSessionAccessDetailsWithUserID()`
- `GetSessionDetails()` - Line 111: Now uses `GetAccessSessionAccessDetailsWithUserID()`

### Command Layer

#### `pkg/commands/access/actions/use.go`
- Line 41: Now uses `GetAccessSessionWithUserID()` instead of direct API call

#### `pkg/commands/access/actions/reset.go`
- Line 33: Now uses `ResetAccessSessionCredentialsWithUserID()` instead of direct API call
- Line 45: Now uses `GetAccessSessionWithUserID()` instead of direct API call (in polling loop)

## API Endpoints Covered

| Endpoint | Method | Wrapper Function | Used By |
|----------|--------|-----------------|---------|
| `GET /api/client/v1/access-sessions` | GET | `ListAccessSessionsWithUserID` | `access list` |
| `GET /api/client/v1/access-sessions/{id}` | GET | `GetAccessSessionWithUserID` | `access use`, `access reset-credentials` |
| `GET /api/client/v1/access-sessions/{id}/access-details` | GET | `GetAccessSessionAccessDetailsWithUserID` | `access use` |
| `POST /api/client/v1/access-sessions/{id}/reset-credentials` | POST | `ResetAccessSessionCredentialsWithUserID` | `access reset-credentials` |

## Usage Examples

### With Environment Variable

```bash
# Set userId via environment variable
export APONO_USER_ID="8bf6b4b7-f1b0-40bb-a382-111111111111"

# All commands now include user_id in API calls
apono-local access list
apono-local access use postgresql-local-postgres-adce68
apono-local access reset-credentials postgresql-local-postgres-adce68
apono-local access save-creds postgresql-local-postgres-adce68
apono-local access setup-mcp postgresql-local-postgres-adce68
```

### With Profile Config

```bash
# Login stores userId in profile
apono-local login --api-url http://localhost:9010 --personal-token "xxx" --profile local

# Commands automatically use userId from profile
apono-local access list --profile local
apono-local access use postgresql-local-postgres-adce68 --profile local
```

### One-time Override

```bash
# Override profile userId for single command
APONO_USER_ID="different-user-id" apono-local access list --profile local
```

## Backend Query Parameters

The backend will now see these query parameters:

### List Sessions
```
GET /api/client/v1/access-sessions?skip=0&user_id=8bf6b4b7-f1b0-40bb-a382-111111111111
```

### Get Session
```
GET /api/client/v1/access-sessions/session-id-123?user_id=8bf6b4b7-f1b0-40bb-a382-111111111111
```

### Get Session Details
```
GET /api/client/v1/access-sessions/session-id-123/access-details?user_id=8bf6b4b7-f1b0-40bb-a382-111111111111
```

### Reset Credentials
```
POST /api/client/v1/access-sessions/session-id-123/reset-credentials?user_id=8bf6b4b7-f1b0-40bb-a382-111111111111
```

## Testing

### Verify userId is Sent

1. **Check backend logs** - Look for `user_id` in query params
2. **Use network inspection** - Monitor HTTP requests
3. **Test with different users** - Use `APONO_USER_ID` env var

### Test Commands

```bash
cd /Users/dmitryogurko/apono/apono-cli

# Login to local environment
go run ./cmd/apono login \
  --api-url http://localhost:9010 \
  --personal-token "your-token" \
  --profile local

# Test with explicit userId
APONO_USER_ID="test-user-123" go run ./cmd/apono access list --profile local

# Test without explicit userId (uses profile)
go run ./cmd/apono access list --profile local

# Test all commands
APONO_USER_ID="test-user-123" go run ./cmd/apono access use session-id --profile local
APONO_USER_ID="test-user-123" go run ./cmd/apono access reset-credentials session-id --profile local
```

## Validation Checklist

✅ All GET requests to `/api/client/v1/access-sessions` include `user_id`
✅ All GET requests to `/api/client/v1/access-sessions/{id}` include `user_id`
✅ All GET requests to `/api/client/v1/access-sessions/{id}/access-details` include `user_id`
✅ All POST requests to `/api/client/v1/access-sessions/{id}/reset-credentials` include `user_id`
✅ Environment variable `APONO_USER_ID` takes precedence
✅ Profile config userId used as fallback
✅ Graceful fallback to standard API when no userId available
✅ All command actions updated to use wrapper methods
✅ No direct `ClientAPI.AccessSessionsAPI` calls in command layer

## Implementation Details

### URL Construction

All wrapper methods use this pattern:

```go
// Build full URL with scheme and host
scheme := cfg.Scheme
if scheme == "" {
    scheme = "https"
}
host := cfg.Host
fullURL := fmt.Sprintf("%s://%s/api/client/v1/access-sessions/...", scheme, host)

// Parse URL and add query parameter
u, err := url.Parse(fullURL)
if err != nil {
    return nil, nil, fmt.Errorf("failed to parse URL: %w", err)
}

q := u.Query()
q.Add("user_id", userID)
u.RawQuery = q.Encode()
```

### Authentication

All wrapper methods preserve authentication by using the configured HTTP client:

```go
// Execute request (the HTTP client already has auth configured)
httpResp, err := cfg.HTTPClient.Do(httpReq)
```

The HTTP client is configured in `pkg/aponoapi/client_factory.go` with:
- OAuth2 token refresh
- Personal token authentication
- Proper headers and User-Agent

### Error Handling

All wrapper methods:
1. Check for HTTP client errors
2. Read response body
3. Check HTTP status codes
4. Parse JSON responses
5. Return descriptive error messages

## Backwards Compatibility

The implementation is fully backwards compatible:

- If `APONO_USER_ID` is not set and profile has no userId, the standard API client is used
- All existing commands work exactly as before when no userId is available
- The backend can handle requests with or without the `user_id` parameter

## Future Enhancements

Potential improvements:

1. **Add userId to other API endpoints** - Request templates, access groups, etc.
2. **Validate userId format** - Ensure it's a valid UUID before sending
3. **Cache userId** - Store in context to avoid repeated env var lookups
4. **Logging** - Add debug logging when userId is injected
5. **Configuration** - Allow disabling userId injection via config flag

## Related Documentation

- [CREDENTIALS_COMMANDS.md](./CREDENTIALS_COMMANDS.md) - Credentials management commands
- [CLAUDE.md](./CLAUDE.md) - General project documentation
- [README.md](./README.md) - Installation and build instructions
