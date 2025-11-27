# MCP Proxy with Slack Approval Workflow

This MCP (Model Context Protocol) proxy adds security controls and approval workflows for AI interactions with sensitive operations.

## Features

- **Risk Detection**: Automatically detects risky operations based on method names and parameter content
- **Slack Approval Workflow**: Sends approval requests to Slack when risky operations are detected
- **Audit Logging**: Comprehensive logging of all requests with risk assessments and approval decisions
- **Flexible Configuration**: YAML-based configuration with environment variable support

## Quick Start

### 1. Set Up Slack App

1. Create a new Slack app at https://api.slack.com/apps
2. Enable the following **Bot Token Scopes** in OAuth & Permissions:
   - `chat:write` - Send messages
   - `chat:write.public` - Send messages to channels the bot isn't in (only needed for channels)
   - `im:write` - Send direct messages to users (only needed for DMs)
3. Enable **Interactivity** in the app settings:
   - Set Request URL to: `http://your-server:8080/slack/interactions`
   - (Use ngrok for local development: `ngrok http 8080`)
4. Install the app to your workspace
5. Copy the **Bot User OAuth Token** (starts with `xoxb-`)
6. Copy the **Signing Secret** from Basic Information
7. (Optional) To find your User ID: Click your profile in Slack → More → Copy member ID

### 2. Configure the Proxy

Create a `mcp-proxy-config.yaml` file:

```yaml
slack:
  enabled: true
  bot_token: xoxb-your-bot-token-here

  # Choose ONE of the following:
  # Option 1: Send to a channel
  channel_id: C1234567890  # Your Slack channel ID (starts with C)

  # Option 2: Send DM to a specific user (takes precedence over channel_id)
  # user_id: U1234567890  # Your Slack user ID (starts with U)

  signing_secret: your-signing-secret-here
  skip_verification: false  # Optional: Set to true to disable signature verification (debugging only!)
  callback_port: 8011
  timeout: 5m

risk:
  enabled: true
  block_on_risk: false  # Use approval workflow instead of auto-blocking
  risky_methods:
    - delete
    - drop
    - truncate
  risky_keywords:
    - "DROP TABLE"
    - "DELETE FROM"

# Proxy configuration - Choose HTTP or STDIO mode
proxy:
  name: "Postgres MCP"
  command: "docker"
  args: [run, -i, --rm, -e, DATABASE_URI, crystaldba/postgres-mcp, --access-mode=unrestricted]
  env:
    DATABASE_URI: "postgresql://postgres:postgres@localhost:5432/mydb"
```

### 3. Run the Proxy

```bash
# Configuration file is required
apono-cli mcp-proxy --config mcp-proxy-config.yaml
```

With environment variables:

```bash
export SLACK_BOT_TOKEN="xoxb-your-token"
export SLACK_SIGNING_SECRET="your-secret"
apono-cli mcp-proxy --config mcp-proxy-config.yaml
```

**Note:** The mode (HTTP vs STDIO) is automatically detected from your config:
- If `proxy.command` is set → STDIO subprocess mode
- If `proxy.endpoint` is set → HTTP proxy mode
- You can override with `--mode stdio` or `--mode http` flags

## How It Works

```
┌─────────────┐
│   AI/LLM    │
│   Client    │
└──────┬──────┘
       │ MCP Request
       ▼
┌─────────────────────────────────────────┐
│           MCP Proxy                     │
│                                         │
│  ┌─────────────────┐                   │
│  │ Risk Detection  │                   │
│  └────────┬────────┘                   │
│           │                             │
│           ▼                             │
│    Risky Operation?                     │
│           │                             │
│      Yes  │                             │
│           ▼                             │
│  ┌─────────────────┐                   │
│  │ Slack Notifier  │─────────┐         │
│  └─────────────────┘         │         │
│                               │         │
└───────────────────────────────┼─────────┘
                                │
                                ▼
                         ┌──────────────┐
                         │    Slack     │
                         │   Channel    │
                         └──────┬───────┘
                                │
                         User Clicks
                         Approve/Deny
                                │
                                ▼
                         ┌──────────────┐
                         │   Callback   │
                         │   Handler    │
                         └──────┬───────┘
                                │
                                ▼
                         Decision Applied
```

### Workflow Steps

1. **Request arrives** - AI client sends MCP request to proxy
2. **Risk detection** - Proxy analyzes method names and parameters
3. **Slack notification** - If risky, sends approval request to Slack channel
4. **Wait for approval** - Proxy blocks and waits for user response
5. **User responds** - User clicks Approve ✅ or Deny ❌ button in Slack
6. **Decision applied**:
   - **Approved**: Request forwarded to actual MCP server
   - **Denied/Timeout**: Error returned to AI client
7. **Audit logged** - Complete record saved with approval metadata

## Configuration Reference

### Proxy Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | No | Display name for the proxy |
| `endpoint` | string | HTTP mode | HTTP endpoint to proxy requests to |
| `headers` | map | No | HTTP headers to add to requests |
| `command` | string | STDIO mode | Command to execute for subprocess |
| `args` | []string | STDIO mode | Arguments for the command |
| `env` | map | STDIO mode | Environment variables for the subprocess |

**HTTP Mode Example:**
```yaml
proxy:
  name: "MCP HTTP Proxy"
  endpoint: "http://localhost:3000/mcp"
  headers:
    Authorization: "Bearer my-token"
```

**STDIO Mode Example:**
```yaml
proxy:
  name: "Postgres MCP"
  command: "docker"
  args: [run, -i, --rm, -e, DATABASE_URI, crystaldba/postgres-mcp]
  env:
    DATABASE_URI: "postgresql://postgres:postgres@localhost:5432/mydb"
```

### Slack Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `enabled` | boolean | Yes | Enable Slack approval workflow |
| `bot_token` | string | Yes | Slack bot token (xoxb-...) |
| `channel_id` | string | Yes | Slack channel ID for notifications |
| `signing_secret` | string | Yes | Slack signing secret for verification |
| `callback_port` | int | Yes | Port for callback server (default: 8080) |
| `timeout` | duration | Yes | Approval timeout (e.g., 5m, 10m, 1h) |

### Risk Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `enabled` | boolean | Yes | Enable risk detection |
| `block_on_risk` | boolean | Yes | Auto-block (true) or use approval (false) |
| `risky_methods` | []string | No | Method name patterns to flag as risky |
| `risky_keywords` | []string | No | Keywords in parameters to flag |
| `allowed_methods` | []string | No | Whitelist of safe methods |

### Environment Variables

All configuration values support environment variable substitution:

```yaml
slack:
  bot_token: ${SLACK_BOT_TOKEN}
  channel_id: ${SLACK_CHANNEL_ID}
  signing_secret: ${SLACK_SIGNING_SECRET}
```

## Audit Logs

Audit logs are stored in JSON format with complete request details:

```json
{
  "timestamp": "2025-11-27T10:30:00Z",
  "method": "tools/call",
  "client_name": "Claude Desktop",
  "params": {...},
  "risk": {
    "is_risky": true,
    "level": 3,
    "reason": "Detected dangerous operation in parameters",
    "matched_rule": "keyword:DROP TABLE"
  },
  "approval_requested": true,
  "approved": true,
  "approved_by": "john.doe",
  "approved_at": "2025-11-27T10:31:30Z",
  "blocked": false
}
```

## Slack Message Format

When a risky operation is detected, a message is sent to Slack:

```
🚨 Risky MCP Operation Detected

Method: tools/call
Risk Level: HIGH
Client: Claude Desktop
Reason: Detected DELETE operation

Matched Rule: keyword:DELETE FROM

Parameters:
{
  "sql": "DELETE FROM users WHERE id = 123"
}

Time: 2025-11-27T10:30:00Z

[✅ Approve]  [❌ Deny]
```

After a decision is made, the message is updated:

```
✅ Request APPROVED

Decision by: john.doe
Time: 2025-11-27T10:31:30Z

Status: APPROVED
```

## Security Considerations

1. **Slack Signature Verification**: All incoming Slack requests are verified using HMAC-SHA256
2. **Timeout Protection**: Requests auto-deny after the configured timeout
3. **First Responder Wins**: Only the first approval/denial is accepted
4. **Audit Trail**: All decisions are logged with timestamps and approver identity
5. **TLS Required**: Use HTTPS/TLS for production callback URLs

## Development & Testing

### Run Tests

```bash
# Test approval workflow
go test ./pkg/commands/mcp-proxy/approval/... -v

# Test configuration
go test ./pkg/commands/mcp-proxy/config/... -v
```

### Local Development with ngrok

1. Start the proxy:
   ```bash
   apono-cli mcp-proxy --config mcp-proxy-config.yaml
   ```

2. In another terminal, start ngrok:
   ```bash
   ngrok http 8080
   ```

3. Update Slack app Interactivity URL with ngrok URL:
   ```
   https://abc123.ngrok.io/slack/interactions
   ```

## Troubleshooting

### Slack callback not working

1. Verify Interactivity is enabled in Slack app settings
2. Check that callback URL is accessible from the internet
3. Ensure signing secret matches in config
4. Check logs for signature verification errors

### Requests timing out

1. Increase timeout in config: `timeout: 10m`
2. Ensure Slack notifications are being sent
3. Check that callback server is running on correct port
4. Verify firewall rules allow incoming connections

### Approval not being applied

1. Check that approval store is receiving updates
2. Verify Done channel is being signaled
3. Check for race conditions in logs
4. Ensure only one proxy instance is running

## Architecture

### Packages

- **`approval/`** - Approval workflow orchestration
  - `approval_manager.go` - Core approval logic
  - `approval_store.go` - In-memory approval storage
  - `types.go` - Shared type definitions

- **`notifier/`** - Slack integration
  - `slack_notifier.go` - Slack API client
  - `slack_callback_handler.go` - HTTP handler for Slack interactions
  - `slack_callback_server.go` - Embedded HTTP server

- **`config/`** - Configuration management
  - `config.go` - YAML loading and validation

- **`auditor/`** - Risk detection and audit logging
  - `risk_aware_auditor.go` - Integration point
  - `pattern_risk_detector.go` - Risk detection logic

### Key Interfaces

```go
// Approval Manager
type ApprovalManager interface {
    RequestApproval(ctx context.Context, req RequestAudit) (bool, *ApprovalResponse, error)
}

// Slack Notifier
type SlackNotifier interface {
    SendApprovalRequest(ctx context.Context, req ApprovalRequest) (string, error)
}

// Approval Store
type ApprovalStore interface {
    CreatePending(approvalID string, request PendingApproval) error
    GetPending(approvalID string) (*PendingApproval, error)
    UpdateResponse(approvalID string, response ApprovalResponse) error
    DeletePending(approvalID string) error
}
```

## Troubleshooting

### Slack "Invalid Signature" Errors

If you're getting "Invalid Slack signature" errors, try these steps:

1. **Verify your signing secret**: Make sure the `signing_secret` in your config exactly matches the one in your Slack app settings (Basic Information → App Credentials → Signing Secret)

2. **Check timestamp sync**: The signature verification requires your server's clock to be within 5 minutes of Slack's servers. Ensure your system time is accurate.

3. **Temporary bypass for debugging**: You can temporarily disable signature verification to isolate the issue:
   ```yaml
   slack:
     skip_verification: true  # NOT recommended for production!
   ```
   ⚠️ **WARNING**: Only use `skip_verification: true` for local debugging. Never use this in production as it disables security verification.

4. **Enable debug logging**: The proxy includes detailed debug logging for signature verification. Check your logs for detailed information about what's failing.

### Slack Interactive Messages Not Working

1. Ensure **Interactivity** is enabled in your Slack app settings
2. Verify the **Request URL** is set to `http://your-server:8080/slack/interactions`
3. For local development, use ngrok and update the Request URL to your ngrok URL
4. Check that the callback server is running (look for "Starting Slack callback server" in logs)

## Future Enhancements

- [ ] Redis-based approval store for distributed deployments
- [ ] Multiple approver requirements
- [ ] Designated approver lists
- [ ] Custom approval rules per operation type
- [ ] Integration with other chat platforms (Teams, Discord)
- [ ] Web UI for approval management
- [ ] Approval history and analytics dashboard

## License

See main repository LICENSE file.

