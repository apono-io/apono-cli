# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Apono CLI is a command-line interface for managing access permissions to services, databases, and applications. Built in Go, it provides both interactive and non-interactive modes for requesting, viewing, and using access sessions. The CLI also includes an MCP (Model Context Protocol) server for AI integration.

## Build and Development Commands

### Full Build Pipeline
```bash
make all          # Complete build: mod, inst, gen, build, spell, lint, test
make ci           # CI pipeline: all + git diff check
```

### Individual Commands
```bash
make mod          # Run go mod tidy
make inst         # Install required tools (misspell, oapi-codegen, golangci-lint, goreleaser)
make gen          # Run go generate ./...
make build        # Build with goreleaser (creates dist/ directory)
make lint         # Run golangci-lint with auto-fix
make test         # Run tests with coverage (generates dist/coverage.out and dist/coverage.html)
make spell        # Run misspell checker
make clean        # Remove all build artifacts
```

### Testing
```bash
go test -race -covermode=atomic -coverprofile=dist/coverage.out -coverpkg=./... ./...
go tool cover -html=dist/coverage.out -o dist/coverage.html  # View coverage report
```

### Build for Single Target
```bash
goreleaser build --clean --single-target --snapshot
```

### Generate Shell Completions and Manpages
```bash
make manpage      # Generate man pages to contrib/manpage/
make completions  # Generate shell completions (bash, powershell, zsh) to contrib/completion/
```

## Architecture

### Entry Point and Command Structure

- **Entry point**: `cmd/apono/main.go` creates a Runner with version info and executes with signal handling
- **Command configuration**: Uses the Configurator pattern where each command package implements `Configurator` interface
- **Runner initialization** (`pkg/commands/apono/runner.go`): Registers all configurators:
  - `auth.Configurator` - Authentication commands
  - `integrations.Configurator` - Integration management
  - `requests.Configurator` - Access request management
  - `access.Configurator` - Access session management
  - `mcp.Configurator` - MCP server command
- **Command groups**: Defined in `pkg/groups/` (auth, management, other) for organizing command help output

### Core Package Structure

- **pkg/commands/**: Command implementations organized by domain (access, auth, integrations, requests, mcp)
  - Each has a `configurator.go` that registers commands with Cobra
  - Actions subdirectory contains actual command logic
- **pkg/aponoapi/**: API client wrapper with OAuth2 token handling
  - Uses custom `refreshableTokenSource` for automatic token refresh
  - Supports both OAuth2 and personal token authentication
- **pkg/clientapi/**: Generated API client models (likely from OpenAPI spec via oapi-codegen)
- **pkg/config/**: Configuration management for profiles, tokens, and authentication
  - Stores profiles with OAuth2 tokens or personal tokens
  - Default API URL: `https://api.apono.io`
- **pkg/interactive/**: Interactive terminal UI components built with Bubble Tea
  - `flows/` - Multi-step interactive workflows
  - `selectors/` - Interactive selection components (bundles, sessions, integrations, etc.)
  - `inputs/` - Custom input widgets (list_select, text_input, request_loader)
- **pkg/services/**: Business logic for sessions, bundles, and access management
- **pkg/analytics/**: Command usage analytics with command ID tracking
- **pkg/utils/**: Shared utilities for formatting, flags, strings, API helpers
- **pkg/styles/**: Terminal styling and color definitions

### MCP Server Implementation

The MCP command (`pkg/commands/mcp/actions/mcp.go`) runs a stdio-based MCP server:
- Reads JSON-RPC requests from stdin
- Proxies requests to Apono backend at `/api/client/v1/mcp`
- Handles authentication via OAuth2 or personal tokens
- Logs to a separate file (via `utils.InitMcpLogFile()`)
- Uses channels for goroutine coordination to handle SIGTERM gracefully
- Extracts client name from initialize request and sets User-Agent header
- Returns proper JSON-RPC error codes for auth/authorization failures

### Authentication Flow

1. OAuth2 flow via `pkg/aponoapi/` using `golang.org/x/oauth2` and `int128/oauth2cli`
2. Tokens stored in config profiles (`pkg/config/`)
3. Personal token support as alternative auth method
4. Refreshable token source automatically refreshes expired tokens
5. Profile management via `--profile` flag on all commands
6. Context-based client injection via `PersistentPreRunE` in root command

### Interactive Flows

The CLI provides rich interactive experiences using Bubble Tea:
- Session selection with filtering
- Bundle/integration/resource browsing
- Multiple connection methods per session (execute vs print instructions)
- Custom instruction messages for access sessions
- Credential reset suggestions

### Code Generation

- Uses `go generate` (see `make gen`)
- API client models likely generated from OpenAPI spec using `oapi-codegen`
- Build info injected via ldflags in `.goreleaser.yaml`:
  - Version, Commit, Date into `pkg/build` package

### Release Process

Uses GoReleaser (`.goreleaser.yaml`):
- Multi-platform builds (Linux, macOS, Windows, OpenBSD)
- Multi-architecture (amd64, arm, arm64)
- Packages: deb, rpm, Homebrew tap, Scoop bucket
- Includes shell completions and man pages in releases
- GPG signing of checksums

## Development Notes

### Working with Commands

When adding new commands:
1. Create command package under `pkg/commands/<domain>/`
2. Implement `Configurator` interface in `configurator.go`
3. Add action implementations in `actions/` subdirectory
4. Register configurator in `pkg/commands/apono/runner.go`
5. Add to appropriate command group from `pkg/groups/`

### Working with API Client

- Client creation: `aponoapi.CreateClient(ctx, profileName)`
- Client retrieval from context: `aponoapi.GetClient(ctx)`
- Context injection happens in root command's `PersistentPreRunE`
- All commands receive pre-configured client via context

### Interactive Components

- Use `pkg/interactive/selectors/` for building selection UIs
- Use `pkg/interactive/flows/` for multi-step workflows
- Components built with Bubble Tea framework (`charmbracelet/bubbletea`)
- Styling via Lipgloss (`charmbracelet/lipgloss`)

### Logging for MCP

- MCP server logs to separate file (not stdout/stderr)
- Use `utils.McpLogf()` for MCP-related logging
- Regular stdout/stderr used for JSON-RPC protocol communication

## Important Conventions

- All persistent flags handled by root command
- Context used for passing client, version info, analytics, and profile name
- Error handling: commands return errors, runner handles printing
- Profile-based configuration allows multiple account management
- OAuth2 refresh token automatically persisted on refresh
