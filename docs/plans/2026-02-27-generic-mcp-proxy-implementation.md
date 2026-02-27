# Generic MCP Proxy Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace hardcoded backend type configs with a YAML-driven MCP server registry, add Go template credential building, and implement a session watcher that auto-spawns/kills MCP backends when Apono sessions are granted/expire.

**Architecture:** Smart Gateway pattern. A local YAML config (`~/.apono/mcp-servers.yaml`) defines MCP server types with credential templates. A Session Watcher polls Apono sessions and auto-spawns matching backends. Existing `STDIOBackend`, `ProxyManager`, and `MCPHandler` are reused with targeted modifications.

**Tech Stack:** Go 1.20, `text/template` (stdlib), `gopkg.in/yaml.v3` (already in deps), existing MCP proxy infrastructure.

---

## Task 1: MCP Server Registry — Config Types and YAML Loader

**Files:**
- Create: `pkg/commands/mcp/registry/types.go`
- Create: `pkg/commands/mcp/registry/loader.go`
- Create: `pkg/commands/mcp/registry/loader_test.go`

**Step 1: Write the test for loading MCP server config from YAML**

```go
// pkg/commands/mcp/registry/loader_test.go
package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMCPServersConfig_ValidFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "mcp-servers.yaml")

	yaml := `mcp_servers:
  - id: postgres
    name: "PostgreSQL MCP"
    integration_types: ["postgresql", "postgres", "rds-postgresql"]
    command: "npx"
    args: ["-y", "@anthropic-ai/postgres-mcp-server"]
    credential_builder:
      database_url: "postgresql://{{.username}}:{{.password}}@{{.host}}:{{.port}}/{{.db_name}}?sslmode=require"
    env_mapping:
      database_url: "DATABASE_URL"
    arg_mapping:
      - database_url
`
	if err := os.WriteFile(configPath, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	registry, err := LoadMCPServersConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(registry.Servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(registry.Servers))
	}

	s := registry.Servers[0]
	if s.ID != "postgres" {
		t.Errorf("expected id 'postgres', got %q", s.ID)
	}
	if s.Command != "npx" {
		t.Errorf("expected command 'npx', got %q", s.Command)
	}
	if len(s.IntegrationTypes) != 3 {
		t.Errorf("expected 3 integration types, got %d", len(s.IntegrationTypes))
	}
	if s.CredentialBuilder["database_url"] == "" {
		t.Error("expected credential_builder.database_url to be set")
	}
	if s.EnvMapping["database_url"] != "DATABASE_URL" {
		t.Errorf("expected env_mapping database_url=DATABASE_URL, got %q", s.EnvMapping["database_url"])
	}
	if len(s.ArgMapping) != 1 || s.ArgMapping[0] != "database_url" {
		t.Errorf("expected arg_mapping [database_url], got %v", s.ArgMapping)
	}
}

func TestLoadMCPServersConfig_FileNotFound(t *testing.T) {
	_, err := LoadMCPServersConfig("/nonexistent/path.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLookupByIntegrationType(t *testing.T) {
	registry := &MCPServersConfig{
		Servers: []MCPServerDefinition{
			{
				ID:               "postgres",
				IntegrationTypes: []string{"postgresql", "postgres", "rds-postgresql"},
				Command:          "npx",
			},
		},
	}

	s, ok := registry.LookupByIntegrationType("rds-postgresql")
	if !ok {
		t.Fatal("expected to find server for rds-postgresql")
	}
	if s.ID != "postgres" {
		t.Errorf("expected postgres, got %q", s.ID)
	}

	_, ok = registry.LookupByIntegrationType("mysql")
	if ok {
		t.Fatal("expected no match for mysql")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/commands/mcp/registry/ -v -run TestLoad`
Expected: FAIL — package does not exist

**Step 3: Write the types**

```go
// pkg/commands/mcp/registry/types.go
package registry

// MCPServerDefinition defines how to spawn and configure an MCP server
// for a given integration type. Loaded from ~/.apono/mcp-servers.yaml.
type MCPServerDefinition struct {
	ID                string            `yaml:"id"`
	Name              string            `yaml:"name"`
	IntegrationTypes  []string          `yaml:"integration_types"`
	Command           string            `yaml:"command"`
	Args              []string          `yaml:"args,omitempty"`
	CredentialBuilder map[string]string `yaml:"credential_builder,omitempty"`
	EnvMapping        map[string]string `yaml:"env_mapping,omitempty"`
	ArgMapping        []string          `yaml:"arg_mapping,omitempty"`
}

// MCPServersConfig is the top-level config loaded from mcp-servers.yaml.
type MCPServersConfig struct {
	Servers []MCPServerDefinition `yaml:"mcp_servers"`
}

// LookupByIntegrationType finds the first server definition matching
// the given Apono integration type string.
func (c *MCPServersConfig) LookupByIntegrationType(integrationType string) (MCPServerDefinition, bool) {
	for _, s := range c.Servers {
		for _, it := range s.IntegrationTypes {
			if it == integrationType {
				return s, true
			}
		}
	}
	return MCPServerDefinition{}, false
}

// LookupByID finds a server definition by its ID.
func (c *MCPServersConfig) LookupByID(id string) (MCPServerDefinition, bool) {
	for _, s := range c.Servers {
		if s.ID == id {
			return s, true
		}
	}
	return MCPServerDefinition{}, false
}
```

**Step 4: Write the loader**

```go
// pkg/commands/mcp/registry/loader.go
package registry

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadMCPServersConfig reads and parses an mcp-servers.yaml file.
func LoadMCPServersConfig(path string) (*MCPServersConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP servers config: %w", err)
	}

	var config MCPServersConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse MCP servers config: %w", err)
	}

	return &config, nil
}
```

**Step 5: Run tests to verify they pass**

Run: `go test ./pkg/commands/mcp/registry/ -v`
Expected: PASS

**Step 6: Commit**

```bash
git add pkg/commands/mcp/registry/
git commit -m "feat(mcp): add MCP server registry types and YAML loader"
```

---

## Task 2: Credential Builder — Template Rendering

**Files:**
- Create: `pkg/commands/mcp/registry/credential_builder.go`
- Create: `pkg/commands/mcp/registry/credential_builder_test.go`

**Step 1: Write the failing tests**

```go
// pkg/commands/mcp/registry/credential_builder_test.go
package registry

import (
	"testing"
)

func TestBuildCredentials_WithTemplates(t *testing.T) {
	def := MCPServerDefinition{
		CredentialBuilder: map[string]string{
			"database_url": "postgresql://{{.username}}:{{.password}}@{{.host}}:{{.port}}/{{.db_name}}?sslmode=require",
		},
	}

	rawFields := map[string]string{
		"host":     "db.prod.com",
		"port":     "5432",
		"username": "app_user",
		"password": "secret123",
		"db_name":  "mydb",
	}

	result, err := BuildCredentials(def, rawFields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "postgresql://app_user:secret123@db.prod.com:5432/mydb?sslmode=require"
	if result["database_url"] != expected {
		t.Errorf("expected %q, got %q", expected, result["database_url"])
	}
}

func TestBuildCredentials_NoTemplates_Passthrough(t *testing.T) {
	def := MCPServerDefinition{} // no credential_builder

	rawFields := map[string]string{
		"host":     "db.prod.com",
		"password": "secret",
	}

	result, err := BuildCredentials(def, rawFields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["host"] != "db.prod.com" {
		t.Errorf("expected passthrough of host")
	}
	if result["password"] != "secret" {
		t.Errorf("expected passthrough of password")
	}
}

func TestBuildCredentials_MissingField(t *testing.T) {
	def := MCPServerDefinition{
		CredentialBuilder: map[string]string{
			"database_url": "postgresql://{{.username}}:{{.password}}@{{.host}}:{{.port}}/{{.db_name}}",
		},
	}

	rawFields := map[string]string{
		"host": "db.prod.com",
		// missing username, password, port, db_name
	}

	_, err := BuildCredentials(def, rawFields)
	if err == nil {
		t.Fatal("expected error for missing template fields")
	}
}

func TestBuildCredentials_URLEncode(t *testing.T) {
	def := MCPServerDefinition{
		CredentialBuilder: map[string]string{
			"database_url": "postgresql://{{.username}}:{{urlEncode .password}}@{{.host}}:{{.port}}/{{.db_name}}",
		},
	}

	rawFields := map[string]string{
		"host":     "db.prod.com",
		"port":     "5432",
		"username": "app_user",
		"password": "p@ss/w0rd#123",
		"db_name":  "mydb",
	}

	result, err := BuildCredentials(def, rawFields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "postgresql://app_user:p%40ss%2Fw0rd%23123@db.prod.com:5432/mydb"
	if result["database_url"] != expected {
		t.Errorf("expected %q, got %q", expected, result["database_url"])
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./pkg/commands/mcp/registry/ -v -run TestBuildCredentials`
Expected: FAIL — `BuildCredentials` undefined

**Step 3: Implement BuildCredentials**

```go
// pkg/commands/mcp/registry/credential_builder.go
package registry

import (
	"bytes"
	"fmt"
	"net/url"
	"text/template"
)

// templateFuncs provides helper functions available in credential templates.
var templateFuncs = template.FuncMap{
	"urlEncode": url.QueryEscape,
}

// BuildCredentials renders credential templates from an MCPServerDefinition
// against raw session fields. If no CredentialBuilder is defined, raw fields
// are passed through as-is.
func BuildCredentials(def MCPServerDefinition, rawFields map[string]string) (map[string]string, error) {
	if len(def.CredentialBuilder) == 0 {
		result := make(map[string]string, len(rawFields))
		for k, v := range rawFields {
			result[k] = v
		}
		return result, nil
	}

	result := make(map[string]string, len(def.CredentialBuilder))
	for key, tmplStr := range def.CredentialBuilder {
		tmpl, err := template.New(key).Funcs(templateFuncs).Option("missingkey=error").Parse(tmplStr)
		if err != nil {
			return nil, fmt.Errorf("invalid template for credential %q: %w", key, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, rawFields); err != nil {
			return nil, fmt.Errorf("failed to render credential %q: %w", key, err)
		}

		result[key] = buf.String()
	}

	return result, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./pkg/commands/mcp/registry/ -v -run TestBuildCredentials`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/commands/mcp/registry/
git commit -m "feat(mcp): add Go template credential builder"
```

---

## Task 3: Wire Registry into ProxyManager — Replace Hardcoded BackendTypes

This task replaces `defaultBackendTypes()` and `targets.BackendTypeConfig` with the new registry-based config inside the proxy manager.

**Files:**
- Modify: `pkg/commands/mcp/proxy/proxy_manager.go`
- Modify: `pkg/commands/mcp/actions/mcp.go`
- Modify: `pkg/commands/mcp/targets/types.go`

**Step 1: Update LocalProxyManagerConfig to accept MCPServersConfig**

In `pkg/commands/mcp/proxy/proxy_manager.go`, replace the `BackendTypes` field:

```go
// OLD (remove):
// backendTypes map[string]targets.BackendTypeConfig

// NEW:
import "github.com/apono-io/apono-cli/pkg/commands/mcp/registry"

// In LocalProxyManagerConfig:
// OLD: BackendTypes []targets.BackendTypeConfig
// NEW: MCPRegistry *registry.MCPServersConfig

// In LocalProxyManager struct:
// OLD: backendTypes map[string]targets.BackendTypeConfig
// NEW: mcpRegistry *registry.MCPServersConfig
```

Update `NewLocalProxyManager` to store the registry:
```go
// OLD:
// backendTypes: make(map[string]targets.BackendTypeConfig),
// for _, bt := range cfg.BackendTypes {
//     pm.backendTypes[bt.ID] = bt
// }

// NEW:
mcpRegistry: cfg.MCPRegistry,
```

**Step 2: Update InitTarget to use registry + credential builder**

In `proxy_manager.go`, update `InitTarget()`:

```go
func (m *LocalProxyManager) InitTarget(ctx context.Context, targetID string) error {
	// ... (existing access + target loading code stays the same until backend type lookup) ...

	// OLD: backendType, ok := m.backendTypes[target.Type]
	// NEW: look up by target.Type in the registry
	serverDef, ok := m.mcpRegistry.LookupByID(target.Type)
	if !ok {
		// Also try integration type
		serverDef, ok = m.mcpRegistry.LookupByIntegrationType(target.Type)
		if !ok {
			return fmt.Errorf("no MCP server configured for type %q", target.Type)
		}
	}

	// Build credentials using template engine
	credentials, err := registry.BuildCredentials(serverDef, target.Credentials)
	if err != nil {
		return fmt.Errorf("failed to build credentials for %q: %w", targetID, err)
	}

	// Build env vars from composed credentials
	env := make(map[string]string)
	for credKey, envVar := range serverDef.EnvMapping {
		if credValue, ok := credentials[credKey]; ok {
			env[envVar] = credValue
		}
	}

	// Build args from composed credentials
	args := make([]string, len(serverDef.Args))
	copy(args, serverDef.Args)
	for _, credKey := range serverDef.ArgMapping {
		if credValue, ok := credentials[credKey]; ok {
			args = append(args, credValue)
		}
	}

	// Create and start backend (same as before but using serverDef.Command)
	stdioBackend := NewSTDIOBackend(STDIOBackendConfig{
		ID:      targetID,
		Name:    target.Name,
		Type:    serverDef.ID,
		Command: serverDef.Command,
		Args:    args,
		Env:     env,
	})
	// ... rest is the same ...
}
```

**Step 3: Update `supportedTypes()` to use registry**

```go
// OLD:
func (m *LocalProxyManager) supportedTypes() []string {
	types := make([]string, 0, len(m.backendTypes))
	for t := range m.backendTypes {
		types = append(types, t)
	}
	return types
}

// NEW:
func (m *LocalProxyManager) supportedTypes() []string {
	types := make([]string, 0, len(m.mcpRegistry.Servers))
	for _, s := range m.mcpRegistry.Servers {
		types = append(types, s.ID)
	}
	return types
}
```

**Step 4: Update mcp.go to load registry from YAML**

In `pkg/commands/mcp/actions/mcp.go`:

```go
// Add new flag:
var mcpServersFile string
const mcpServersFileFlagName = "mcp-servers-file"
// flags.StringVar(&mcpServersFile, mcpServersFileFlagName, "", "Path to mcp-servers.yaml (default: ~/.apono/mcp-servers.yaml)")

// In runLocalSTDIOServerWithProxy():
// Replace defaultBackendTypes() call with registry loading:

// Resolve MCP servers config path
if mcpServersFile == "" {
	homeDir, _ := os.UserHomeDir()
	mcpServersFile = filepath.Join(homeDir, ".apono", "mcp-servers.yaml")
}

mcpRegistry, err := registry.LoadMCPServersConfig(mcpServersFile)
if err != nil {
	// Fall back to default embedded config if file doesn't exist
	utils.McpLogf("MCP servers config not found at %s, using defaults", mcpServersFile)
	mcpRegistry = DefaultMCPRegistry()
}

// Pass to proxy manager:
pm := proxy.NewLocalProxyManager(proxy.LocalProxyManagerConfig{
	MCPRegistry:     mcpRegistry,
	// ... rest unchanged ...
})
```

**Step 5: Add DefaultMCPRegistry fallback**

In `pkg/commands/mcp/actions/mcp.go`, replace `defaultBackendTypes()`:

```go
// Replace defaultBackendTypes() with:
func DefaultMCPRegistry() *registry.MCPServersConfig {
	return &registry.MCPServersConfig{
		Servers: []registry.MCPServerDefinition{
			{
				ID:               "postgres",
				Name:             "PostgreSQL MCP",
				IntegrationTypes: []string{"postgresql", "postgres", "rds-postgresql"},
				Command:          "npx",
				Args:             []string{"-y", "@modelcontextprotocol/server-postgres"},
				CredentialBuilder: map[string]string{
					"database_url": "postgresql://{{.username}}:{{urlEncode .password}}@{{.host}}:{{.port}}/{{.db_name}}?sslmode=require",
				},
				ArgMapping: []string{"database_url"},
			},
		},
	}
}
```

**Step 6: Remove `targets.BackendTypeConfig` if no longer used elsewhere**

Check references — if `BackendTypeConfig` is only used by the proxy manager, remove it from `pkg/commands/mcp/targets/types.go`. If `builtin_tools.go` references it, update those references too.

**Step 7: Verify build compiles**

Run: `go build ./...`
Expected: PASS

**Step 8: Run existing tests**

Run: `go test ./pkg/commands/mcp/... -v`
Expected: PASS (no existing tests should break)

**Step 9: Commit**

```bash
git add pkg/commands/mcp/proxy/proxy_manager.go pkg/commands/mcp/actions/mcp.go pkg/commands/mcp/targets/types.go
git commit -m "refactor(mcp): wire YAML registry into proxy manager, replace hardcoded backend types"
```

---

## Task 4: Session Watcher — Auto-Spawn and Auto-Kill

This is the core new component: a goroutine that polls Apono sessions and automatically manages backend lifecycle.

**Files:**
- Create: `pkg/commands/mcp/proxy/session_watcher.go`
- Create: `pkg/commands/mcp/proxy/session_watcher_test.go`

**Step 1: Write the test**

```go
// pkg/commands/mcp/proxy/session_watcher_test.go
package proxy

import (
	"context"
	"testing"
	"time"

	"github.com/apono-io/apono-cli/pkg/commands/mcp/registry"
	"github.com/apono-io/apono-cli/pkg/commands/mcp/targets"
)

// mockTargetSource implements targets.TargetSource for testing
type mockTargetSource struct {
	targets []targets.TargetInfo
	defs    map[string]*targets.TargetDefinition
}

func (m *mockTargetSource) ListTargets(ctx context.Context) ([]targets.TargetInfo, error) {
	return m.targets, nil
}

func (m *mockTargetSource) GetTarget(ctx context.Context, id string) (*targets.TargetDefinition, error) {
	if def, ok := m.defs[id]; ok {
		return def, nil
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockTargetSource) EnsureAccess(ctx context.Context, id string) error {
	return nil
}

func TestSessionWatcher_DetectsNewSession(t *testing.T) {
	source := &mockTargetSource{
		targets: []targets.TargetInfo{
			{ID: "prod-pg", Name: "prod-pg", Type: "postgresql", Status: targets.TargetStatusReady},
		},
		defs: map[string]*targets.TargetDefinition{
			"prod-pg": {
				ID:   "prod-pg",
				Name: "prod-pg",
				Type: "postgres",
				Credentials: map[string]string{
					"host": "localhost", "port": "5432",
					"username": "user", "password": "pass", "db_name": "test",
				},
			},
		},
	}

	reg := &registry.MCPServersConfig{
		Servers: []registry.MCPServerDefinition{
			{
				ID:               "postgres",
				IntegrationTypes: []string{"postgresql"},
				Command:          "echo", // use echo for testing, won't actually run MCP
				Args:             []string{"test"},
			},
		},
	}

	initCalled := make(chan string, 1)

	watcher := NewSessionWatcher(SessionWatcherConfig{
		TargetSource: source,
		MCPRegistry:  reg,
		PollInterval: 50 * time.Millisecond,
		OnNewSession: func(targetID string, serverDef registry.MCPServerDefinition, target *targets.TargetDefinition) {
			initCalled <- targetID
		},
		OnExpiredSession: func(targetID string) {},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	go watcher.Start(ctx)

	select {
	case id := <-initCalled:
		if id != "prod-pg" {
			t.Errorf("expected prod-pg, got %s", id)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for session detection")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/commands/mcp/proxy/ -v -run TestSessionWatcher`
Expected: FAIL — `NewSessionWatcher` undefined

**Step 3: Implement SessionWatcher**

```go
// pkg/commands/mcp/proxy/session_watcher.go
package proxy

import (
	"context"
	"time"

	"github.com/apono-io/apono-cli/pkg/commands/mcp/registry"
	"github.com/apono-io/apono-cli/pkg/commands/mcp/targets"
	"github.com/apono-io/apono-cli/pkg/utils"
)

// SessionWatcherConfig configures the session watcher.
type SessionWatcherConfig struct {
	TargetSource     targets.TargetSource
	MCPRegistry      *registry.MCPServersConfig
	PollInterval     time.Duration
	OnNewSession     func(targetID string, serverDef registry.MCPServerDefinition, target *targets.TargetDefinition)
	OnExpiredSession func(targetID string)
}

// SessionWatcher polls Apono sessions and fires callbacks when sessions
// appear (for auto-spawn) or disappear (for auto-kill).
type SessionWatcher struct {
	cfg          SessionWatcherConfig
	knownReady   map[string]bool // targetID -> was ready last poll
}

// NewSessionWatcher creates a new session watcher.
func NewSessionWatcher(cfg SessionWatcherConfig) *SessionWatcher {
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 10 * time.Second
	}
	return &SessionWatcher{
		cfg:        cfg,
		knownReady: make(map[string]bool),
	}
}

// Start begins the polling loop. Blocks until ctx is cancelled.
func (w *SessionWatcher) Start(ctx context.Context) {
	utils.McpLogf("[SessionWatcher] Starting with poll interval %v", w.cfg.PollInterval)

	// Run immediately on start, then on ticker
	w.poll(ctx)

	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			utils.McpLogf("[SessionWatcher] Stopping")
			return
		case <-ticker.C:
			w.poll(ctx)
		}
	}
}

func (w *SessionWatcher) poll(ctx context.Context) {
	targetList, err := w.cfg.TargetSource.ListTargets(ctx)
	if err != nil {
		utils.McpLogf("[SessionWatcher] Error listing targets: %v", err)
		return
	}

	currentReady := make(map[string]bool)

	for _, t := range targetList {
		if t.Status != targets.TargetStatusReady {
			continue
		}

		// Check if this target type has a matching MCP server
		_, hasMatch := w.cfg.MCPRegistry.LookupByIntegrationType(t.Type)
		if !hasMatch {
			continue
		}

		currentReady[t.ID] = true

		// New session detected
		if !w.knownReady[t.ID] {
			utils.McpLogf("[SessionWatcher] New ready target detected: %s (type: %s)", t.ID, t.Type)
			serverDef, _ := w.cfg.MCPRegistry.LookupByIntegrationType(t.Type)

			target, err := w.cfg.TargetSource.GetTarget(ctx, t.ID)
			if err != nil {
				utils.McpLogf("[SessionWatcher] Error getting target %s: %v", t.ID, err)
				continue
			}

			w.cfg.OnNewSession(t.ID, serverDef, target)
		}
	}

	// Detect expired sessions
	for id := range w.knownReady {
		if !currentReady[id] {
			utils.McpLogf("[SessionWatcher] Target no longer ready: %s", id)
			w.cfg.OnExpiredSession(id)
		}
	}

	w.knownReady = currentReady
}
```

**Step 4: Run tests**

Run: `go test ./pkg/commands/mcp/proxy/ -v -run TestSessionWatcher`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/commands/mcp/proxy/session_watcher.go pkg/commands/mcp/proxy/session_watcher_test.go
git commit -m "feat(mcp): add session watcher for auto-spawn/kill of MCP backends"
```

---

## Task 5: Integrate Session Watcher into ProxyManager

Wire the session watcher into the proxy manager so it auto-spawns backends via `InitTarget` and auto-kills via `StopTarget`.

**Files:**
- Modify: `pkg/commands/mcp/proxy/proxy_manager.go`
- Modify: `pkg/commands/mcp/actions/mcp.go`

**Step 1: Add session watcher field to LocalProxyManager**

In `proxy_manager.go`:

```go
// Add to LocalProxyManager struct:
sessionWatcher *SessionWatcher

// Add to LocalProxyManagerConfig:
PollInterval time.Duration // for session watcher, default 10s
```

**Step 2: Create and start watcher in NewLocalProxyManager**

In `NewLocalProxyManager`, after creating the proxy manager:

```go
// Create session watcher
pm.sessionWatcher = NewSessionWatcher(SessionWatcherConfig{
	TargetSource: cfg.TargetSource,
	MCPRegistry:  cfg.MCPRegistry,
	PollInterval: cfg.PollInterval,
	OnNewSession: func(targetID string, serverDef registry.MCPServerDefinition, target *targets.TargetDefinition) {
		utils.McpLogf("[ProxyManager] Auto-spawning backend for target %s", targetID)
		if err := pm.InitTarget(context.Background(), targetID); err != nil {
			utils.McpLogf("[ProxyManager] Failed to auto-spawn %s: %v", targetID, err)
		}
	},
	OnExpiredSession: func(targetID string) {
		utils.McpLogf("[ProxyManager] Auto-killing backend for expired target %s", targetID)
		if err := pm.StopTarget(context.Background(), targetID); err != nil {
			utils.McpLogf("[ProxyManager] Failed to auto-kill %s: %v", targetID, err)
		}
	},
})
```

**Step 3: Start watcher in mcp.go**

In `runLocalSTDIOServerWithProxy`, after creating the proxy manager:

```go
// Start session watcher in background
watcherCtx, watcherCancel := context.WithCancel(ctx)
defer watcherCancel()
go pm.sessionWatcher.Start(watcherCtx)
```

**Step 4: Update Close to stop watcher**

The watcher stops when its context is cancelled (handled by defer in mcp.go), so no change needed in `Close()`.

**Step 5: Verify build compiles**

Run: `go build ./...`
Expected: PASS

**Step 6: Commit**

```bash
git add pkg/commands/mcp/proxy/proxy_manager.go pkg/commands/mcp/actions/mcp.go
git commit -m "feat(mcp): integrate session watcher into proxy manager for auto-spawn"
```

---

## Task 6: Enrich Tool Responses with Flow Hints

Update existing static tools to include `next_step` hints in their responses, and update tool descriptions with the full flow overview.

**Files:**
- Modify: `pkg/commands/mcp/tools/list_available_resources.go`
- Modify: `pkg/commands/mcp/tools/ask_access_assistant.go`
- Modify: `pkg/commands/mcp/tools/create_access_request.go`
- Modify: `pkg/commands/mcp/tools/get_request_details.go`
- Modify: `pkg/commands/mcp/proxy/builtin_tools.go` (list_targets)

**Step 1: Add flow description constant**

Create a shared constant in `pkg/commands/mcp/tools/tools.go`:

```go
const FlowDescription = `
Typical workflow:
1. list_available_resources - see what integrations exist and their access status (you are here)
2. ask_access_assistant - describe your task to get scoped access recommendations
3. create_access_request - request access with the recommended scope
4. get_request_details - check if your request was approved
Once access is granted, database tools automatically become available via tools/list_changed.
5. list_targets - see connected targets and their available tools`
```

**Step 2: Update list_available_resources**

In the `Description()` method, append the flow description.

In the `Execute()` method, wrap the return value to include `next_step`:

```go
// At end of Execute(), before return:
response := map[string]interface{}{
	"resources": resources, // existing result
	"next_step": "Use ask_access_assistant with a description of your task to determine what access to request.",
}
return response, nil
```

**Step 3: Update create_access_request**

Add to response:
```go
"next_step": "Use get_request_details with the request_id to check approval status. Once approved, database tools will automatically become available."
```

**Step 4: Update get_request_details**

Add conditional hint based on status:
```go
if status == "APPROVED" || status == "ACTIVE" {
	response["next_step"] = "Access granted. Database tools are now available in your tool list. Use list_targets to see connected targets and their tools."
} else {
	response["next_step"] = "Request is still pending. Call get_request_details again to check status."
}
```

**Step 5: Update list_targets in builtin_tools.go**

Add tool listing per target and next_step hint to the `handleListTargets` response.

**Step 6: Update tool descriptions**

For each tool, append the flow overview to its `Description()` return value, with "you are here" on the relevant step.

**Step 7: Verify build compiles**

Run: `go build ./...`
Expected: PASS

**Step 8: Commit**

```bash
git add pkg/commands/mcp/tools/ pkg/commands/mcp/proxy/builtin_tools.go
git commit -m "feat(mcp): add flow hints to tool descriptions and responses"
```

---

## Task 7: Simplify Builtin Tools — Remove init_target and stop_target

Since the session watcher handles lifecycle automatically, remove agent-facing init/stop tools.

**Files:**
- Modify: `pkg/commands/mcp/proxy/builtin_tools.go`

**Step 1: Remove init_target and stop_target from GetTools()**

Remove the tool definitions for `init_target` and `stop_target` from the `GetTools()` method.

**Step 2: Remove handler methods**

Remove `handleInitTarget` and `handleStopTarget` methods.

**Step 3: Remove routing in HandleToolCall**

Remove the `case "init_target":` and `case "stop_target":` cases from `HandleToolCall`.

**Step 4: Enrich list_targets response**

Update `handleListTargets` to include available tools per connected target:

```go
func (h *BuiltinToolsHandler) handleListTargets(ctx context.Context) (interface{}, error) {
	targets, err := h.manager.ListTargets(ctx)
	if err != nil {
		return nil, err
	}

	// Get tools from running backends
	connectedTargets := make([]map[string]interface{}, 0)
	pendingRequests := make([]map[string]interface{}, 0)

	for _, t := range targets {
		if t.Initialized {
			toolNames := make([]string, 0)
			if inst := h.manager.getInstance(t.ID); inst != nil {
				tools, _ := inst.Backend.ListTools(ctx)
				for _, tool := range tools {
					toolNames = append(toolNames, PrefixToolName(t.ID, tool.Name))
				}
			}
			connectedTargets = append(connectedTargets, map[string]interface{}{
				"name":            t.Name,
				"type":            t.Type,
				"status":          "connected",
				"available_tools": toolNames,
			})
		} else if t.Status == "needs_access" {
			pendingRequests = append(pendingRequests, map[string]interface{}{
				"name":   t.Name,
				"status": string(t.Status),
			})
		}
	}

	return map[string]interface{}{
		"connected_targets": connectedTargets,
		"pending_requests":  pendingRequests,
		"next_step":         "You can now use the tools listed above directly.",
	}, nil
}
```

**Step 5: Verify build compiles**

Run: `go build ./...`
Expected: PASS

**Step 6: Commit**

```bash
git add pkg/commands/mcp/proxy/builtin_tools.go
git commit -m "refactor(mcp): remove init_target/stop_target, enrich list_targets with tool listing"
```

---

## Task 8: Remove Hardcoded Credential Building from SessionTargetProvider

Replace the `buildCredentials` / `buildPostgresCredentials` / `buildMySQLCredentials` switch in `session_provider.go` with raw field extraction only. Credential composition now happens in the registry's `BuildCredentials`.

**Files:**
- Modify: `pkg/commands/mcp/targets/session_provider.go`

**Step 1: Simplify GetTarget to return raw fields**

The `GetTarget` method currently calls `buildCredentials()` which does the per-type URL composition. Change it to return raw credential fields instead:

```go
// In GetTarget(), replace the buildCredentials call:
// OLD:
// creds, err := buildCredentials(integrationType, jsonCreds)

// NEW: extract raw fields without composing them
creds := make(map[string]string)
for k, v := range jsonCreds {
	creds[k] = fmt.Sprintf("%v", v)
}
```

**Step 2: Remove buildCredentials, buildPostgresCredentials, buildMySQLCredentials, buildGenericCredentials**

Delete these functions entirely — their job is now handled by `registry.BuildCredentials`.

**Step 3: Keep extractCredentialsFromText as fallback**

The text fallback still extracts individual fields (host, port, etc.) — keep it, but make sure it returns raw fields rather than composed URLs.

**Step 4: Verify build compiles**

Run: `go build ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/commands/mcp/targets/session_provider.go
git commit -m "refactor(mcp): simplify session provider to return raw credential fields"
```

---

## Task 9: Clean Up setup_database in Builtin Tools

The `setup_database` builtin tool currently does its own credential fetching and URL composition. With the session watcher auto-spawning, this tool becomes redundant for the agent. However, keep it for backward compatibility but simplify it to use the new registry.

**Files:**
- Modify: `pkg/commands/mcp/proxy/builtin_tools.go`

**Step 1: Simplify or remove setup_database**

Since session watcher auto-spawns, `setup_database` can be simplified to just trigger an access request and let the watcher handle the rest:

```go
func (h *BuiltinToolsHandler) handleSetupDatabase(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	sessionID, ok := args["session_id"].(string)
	if !ok || sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	// Trigger EnsureAccess which will make the session "ready"
	// The session watcher will then detect it and auto-spawn the backend
	err := h.manager.targetSource.EnsureAccess(ctx, sessionID)
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to ensure access: %v", err)), nil
	}

	return successResult("Access ensured. The MCP server will be automatically spawned. Use list_targets to check status."), nil
}
```

**Step 2: Consider removing setup_database entirely**

If the agent flow is: `create_access_request` → auto-spawn, then `setup_database` is truly redundant. Decide based on whether there are existing integrations depending on it.

For now, keep it as a simplified pass-through to maintain backward compat.

**Step 3: Verify build compiles**

Run: `go build ./...`
Expected: PASS

**Step 4: Commit**

```bash
git add pkg/commands/mcp/proxy/builtin_tools.go
git commit -m "refactor(mcp): simplify setup_database to delegate to session watcher"
```

---

## Task 10: End-to-End Integration Test

**Files:**
- Modify: `pkg/commands/mcp/actions/test.go` (add new test command)

**Step 1: Add e2e test command**

Add a new test subcommand `test-auto-spawn` that:

1. Lists available integrations
2. Picks a postgres integration
3. Creates an access request
4. Waits for approval
5. Verifies tools appear via `tools/list`

This uses the existing test infrastructure in `test.go`.

**Step 2: Manual testing checklist**

- [ ] Start proxy: `apono mcp --proxy`
- [ ] Verify tools/list returns static tools only (no backends yet)
- [ ] Request access to a postgres database via `create_access_request`
- [ ] Wait for approval
- [ ] Verify postgres tools appear automatically (tools/list_changed notification + tools/list)
- [ ] Use a postgres tool (e.g., query)
- [ ] Wait for session to expire (or revoke in Apono UI)
- [ ] Verify postgres tools disappear (tools/list_changed notification)

**Step 3: Commit**

```bash
git add pkg/commands/mcp/actions/test.go
git commit -m "test(mcp): add auto-spawn e2e test command"
```

---

## Task 11: Create Default mcp-servers.yaml Template

**Files:**
- Create: `pkg/commands/mcp/registry/default_config.go`

**Step 1: Write embedded default config**

```go
// pkg/commands/mcp/registry/default_config.go
package registry

// DefaultConfig returns the built-in MCP server definitions.
// Used as fallback when ~/.apono/mcp-servers.yaml doesn't exist.
func DefaultConfig() *MCPServersConfig {
	return &MCPServersConfig{
		Servers: []MCPServerDefinition{
			{
				ID:               "postgres",
				Name:             "PostgreSQL MCP",
				IntegrationTypes: []string{"postgresql", "postgres", "rds-postgresql", "azure-postgresql", "gcp-postgresql"},
				Command:          "npx",
				Args:             []string{"-y", "@modelcontextprotocol/server-postgres"},
				CredentialBuilder: map[string]string{
					"database_url": "postgresql://{{.username}}:{{urlEncode .password}}@{{.host}}:{{.port}}/{{.db_name}}?sslmode=require",
				},
				ArgMapping: []string{"database_url"},
			},
		},
	}
}
```

**Step 2: Update mcp.go to use DefaultConfig**

Replace `DefaultMCPRegistry()` from Task 3 with `registry.DefaultConfig()`.

**Step 3: Commit**

```bash
git add pkg/commands/mcp/registry/default_config.go pkg/commands/mcp/actions/mcp.go
git commit -m "feat(mcp): add default MCP server config with postgres"
```

---

## Summary: Task Dependencies

```
Task 1: Registry types + loader        (independent)
Task 2: Credential builder             (depends on Task 1)
Task 3: Wire registry into proxy       (depends on Tasks 1, 2)
Task 4: Session watcher                (depends on Task 1)
Task 5: Integrate watcher into proxy   (depends on Tasks 3, 4)
Task 6: Enrich tool responses          (independent)
Task 7: Simplify builtin tools         (depends on Task 5)
Task 8: Simplify session provider      (depends on Task 3)
Task 9: Clean up setup_database        (depends on Tasks 5, 7)
Task 10: E2E test                      (depends on all above)
Task 11: Default config                (depends on Task 3)
```

**Parallel tracks:**
- Track A: Tasks 1 → 2 → 3 → 8 → 11
- Track B: Task 4 (can start after Task 1)
- Track C: Task 6 (fully independent)
- Merge: Task 5 (joins A + B) → Task 7 → Task 9
- Final: Task 10
