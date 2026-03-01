package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/apono-io/apono-cli/pkg/commands/mcp/approval"
	"github.com/apono-io/apono-cli/pkg/commands/mcp/registry"
	"github.com/apono-io/apono-cli/pkg/commands/mcp/risk"
	"github.com/apono-io/apono-cli/pkg/commands/mcp/targets"
	"github.com/apono-io/apono-cli/pkg/utils"
)

// ProxyManager is the interface the handler uses to route dynamic tools
type ProxyManager interface {
	ListDynamicTools(ctx context.Context) ([]DynamicToolSchema, error)
	IsDynamicTool(name string) bool
	ExecuteDynamicTool(ctx context.Context, name string, arguments json.RawMessage) (interface{}, error)
	// SetToolsChangedCallback sets a function called when the dynamic tool list changes
	// (e.g., after init_target or stop_target)
	SetToolsChangedCallback(fn func())
	Close() error
}

// BackendInstance represents a running backend for a target
type BackendInstance struct {
	TargetID      string
	TargetName    string
	Type          string
	IntegrationID string // Apono integration ID (for approval requests)
	Backend       Backend
	StartedAt     time.Time
	LastUsed      time.Time
	mu            sync.Mutex
}

func (i *BackendInstance) UpdateLastUsed() {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.LastUsed = time.Now()
}

func (i *BackendInstance) GetLastUsed() time.Time {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.LastUsed
}

// LocalProxyManager implements ProxyManager for a single local user
type LocalProxyManager struct {
	mu               sync.RWMutex
	instances        map[string]*BackendInstance // targetID -> instance
	mcpRegistry      *registry.MCPServersConfig
	targetSource     targets.TargetSource
	riskDetector     risk.RiskDetector
	approver         approval.Approver
	cleanupTimeout   time.Duration
	requestID        int64
	done             chan struct{}
	toolsChangedFn   func() // called when dynamic tool list changes
	apiBaseURL       string       // Apono API base URL
	httpClient       *http.Client // Authenticated HTTP client
	targetsFilePath  string       // Path to targets.yaml file
	sessionWatcher   *SessionWatcher

	builtinTools *BuiltinToolsHandler
}

// LocalProxyManagerConfig configures the local proxy manager
type LocalProxyManagerConfig struct {
	MCPRegistry     *registry.MCPServersConfig
	TargetSource    targets.TargetSource
	RiskDetector    risk.RiskDetector
	Approver        approval.Approver
	CleanupTimeout  time.Duration
	PollInterval    time.Duration
	APIBaseURL      string       // Apono API base URL
	HTTPClient      *http.Client // Authenticated HTTP client
	TargetsFilePath string       // Path to targets.yaml file
}

// NewLocalProxyManager creates a new local proxy manager
func NewLocalProxyManager(cfg LocalProxyManagerConfig) *LocalProxyManager {
	if cfg.CleanupTimeout <= 0 {
		cfg.CleanupTimeout = 30 * time.Minute
	}

	pm := &LocalProxyManager{
		instances:       make(map[string]*BackendInstance),
		mcpRegistry:     cfg.MCPRegistry,
		targetSource:    cfg.TargetSource,
		riskDetector:    cfg.RiskDetector,
		approver:        cfg.Approver,
		cleanupTimeout:  cfg.CleanupTimeout,
		done:            make(chan struct{}),
		apiBaseURL:      cfg.APIBaseURL,
		httpClient:      cfg.HTTPClient,
		targetsFilePath: cfg.TargetsFilePath,
	}

	pm.builtinTools = NewBuiltinToolsHandler(pm)

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

	return pm
}

// SessionWatcher returns the session watcher instance.
func (m *LocalProxyManager) SessionWatcher() *SessionWatcher {
	return m.sessionWatcher
}

// TargetSource returns the target source used by this manager.
func (m *LocalProxyManager) TargetSource() targets.TargetSource {
	return m.targetSource
}

// SetToolsChangedCallback sets a function called when the dynamic tool list changes
func (m *LocalProxyManager) SetToolsChangedCallback(fn func()) {
	m.toolsChangedFn = fn
}

func (m *LocalProxyManager) notifyToolsChanged() {
	if m.toolsChangedFn != nil {
		m.toolsChangedFn()
	}
}

// ListDynamicTools returns all dynamic tools (built-in + backend tools)
func (m *LocalProxyManager) ListDynamicTools(ctx context.Context) ([]DynamicToolSchema, error) {
	var allTools []DynamicToolSchema

	// Add built-in proxy tools
	for _, t := range m.builtinTools.GetTools() {
		allTools = append(allTools, DynamicToolSchema{
			Name:        PrefixToolName(BuiltinBackendID, t.Name),
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}

	// Add tools from active backends
	backendTools, err := m.listToolsFromBackends(ctx)
	if err != nil {
		utils.McpLogf("[ProxyManager] Error listing backend tools: %v", err)
	} else {
		for _, t := range backendTools {
			allTools = append(allTools, DynamicToolSchema{
				Name:        PrefixToolName(t.BackendID, t.Name),
				Description: t.Description,
				InputSchema: t.InputSchema,
			})
		}
	}

	return allTools, nil
}

// IsDynamicTool checks if a tool name belongs to the proxy layer
func (m *LocalProxyManager) IsDynamicTool(name string) bool {
	if !HasNamespacePrefix(name) {
		return false
	}

	backendID, _, err := ParseToolName(name)
	if err != nil {
		return false
	}

	// Check built-in tools
	if backendID == BuiltinBackendID {
		return true
	}

	// Check active backends
	return m.hasBackend(backendID)
}

// ExecuteDynamicTool executes a dynamic tool
func (m *LocalProxyManager) ExecuteDynamicTool(ctx context.Context, name string, arguments json.RawMessage) (interface{}, error) {
	backendID, toolName, err := ParseToolName(name)
	if err != nil {
		return nil, fmt.Errorf("invalid dynamic tool name: %w", err)
	}

	// Handle built-in proxy tools
	if backendID == BuiltinBackendID {
		return m.builtinTools.HandleToolCall(ctx, toolName, arguments)
	}

	// Parse arguments for risk detection
	var args map[string]interface{}
	if arguments != nil {
		if err := json.Unmarshal(arguments, &args); err != nil {
			args = make(map[string]interface{})
		}
	}

	// Risk detection
	if m.riskDetector != nil {
		riskResult := m.riskDetector.DetectRisk(toolName, args)
		if riskResult.IsRisky {
			utils.McpLogf("[ProxyManager] Risk detected for %s: %s", name, riskResult.Reason)

			if m.approver != nil {
				// Look up integration ID from backend instance
				integrationID := ""
				if inst := m.getInstance(backendID); inst != nil {
					integrationID = inst.IntegrationID
				}

				approved, err := m.approver.RequestApproval(ctx, approval.ApprovalRequest{
					ToolName:      toolName,
					Arguments:     args,
					Reason:        riskResult.Reason,
					RiskLevel:     riskLevelToString(riskResult.Level),
					MatchedRule:   riskResult.MatchedRule,
					TargetID:      backendID,
					IntegrationID: integrationID,
				})
				if err != nil {
					return nil, fmt.Errorf("approval request failed: %w", err)
				}
				if !approved {
					return ToolCallResult{
						Content: []ContentItem{
							{Type: "text", Text: fmt.Sprintf("Operation blocked: %s (approval denied)", riskResult.Reason)},
						},
						IsError: true,
					}, nil
				}
			} else {
				// No approver configured, block the request
				return ToolCallResult{
					Content: []ContentItem{
						{Type: "text", Text: fmt.Sprintf("Operation blocked: %s", riskResult.Reason)},
					},
					IsError: true,
				}, nil
			}
		}
	}

	// Route to backend
	backend, modifiedArgs, err := m.routeToolCall(ctx, backendID, args)
	if err != nil {
		return nil, fmt.Errorf("failed to route tool call: %w", err)
	}

	// Build and send the tool call request
	reqID := atomic.AddInt64(&m.requestID, 1)
	reqBytes, err := RebuildToolCallRequest(reqID, toolName, modifiedArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to build tool call request: %w", err)
	}

	respBytes, err := backend.Send(ctx, reqBytes)
	if err != nil {
		return nil, fmt.Errorf("backend error: %w", err)
	}

	// Parse the response and return the result
	var resp JSONRPCResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse backend response: %w", err)
	}

	if resp.Error != nil {
		return ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: fmt.Sprintf("Backend error: %s", resp.Error.Message)},
			},
			IsError: true,
		}, nil
	}

	if resp.Result == nil {
		return ToolCallResult{
			Content: []ContentItem{
				{Type: "text", Text: "No result from backend"},
			},
		}, nil
	}

	// Return the raw result from the backend
	var result interface{}
	if err := json.Unmarshal(*resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}

	return result, nil
}

// Close stops all backends
func (m *LocalProxyManager) Close() error {
	close(m.done)

	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error
	for targetID, instance := range m.instances {
		if err := instance.Backend.Close(); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to close %s: %w", targetID, err)
			}
		}
	}
	m.instances = make(map[string]*BackendInstance)

	return firstErr
}

// ListTargets returns available targets
func (m *LocalProxyManager) ListTargets(ctx context.Context) ([]targets.TargetInfo, error) {
	targetList, err := m.targetSource.ListTargets(ctx)
	if err != nil {
		return nil, err
	}

	// Augment with initialized status
	for i := range targetList {
		targetList[i].Initialized = m.isInitialized(targetList[i].ID)
	}

	return targetList, nil
}

// InitTarget initializes a target backend
func (m *LocalProxyManager) InitTarget(ctx context.Context, targetID string) error {
	// Check if already initialized and healthy
	if m.isInitialized(targetID) {
		instance := m.getInstance(targetID)
		if instance != nil {
			if err := instance.Backend.Health(ctx); err == nil {
				instance.UpdateLastUsed()
				utils.McpLogf("[ProxyManager] Target %s already initialized", targetID)
				return nil
			}
			// Unhealthy, close and reinitialize
			utils.McpLogf("[ProxyManager] Target %s unhealthy, reinitializing", targetID)
			m.stopTargetInternal(targetID)
		}
	}

	// Ensure we have access
	if err := m.targetSource.EnsureAccess(ctx, targetID); err != nil {
		return fmt.Errorf("failed to ensure access for %s: %w", targetID, err)
	}

	// Load target config with credentials
	target, err := m.targetSource.GetTarget(ctx, targetID)
	if err != nil {
		return fmt.Errorf("failed to get target config: %w", err)
	}

	// Look up MCP server definition by ID first, then by integration type
	serverDef, ok := m.mcpRegistry.LookupByID(target.Type)
	if !ok {
		serverDef, ok = m.mcpRegistry.LookupByIntegrationType(target.Type)
		if !ok {
			return fmt.Errorf("no MCP server configured for type %q (supported: %v)", target.Type, m.supportedTypes())
		}
	}

	// Build credentials using the registry's credential builder (template rendering)
	creds, err := registry.BuildCredentials(serverDef, target.Credentials)
	if err != nil {
		return fmt.Errorf("failed to build credentials for %s: %w", targetID, err)
	}

	// Build environment variables from credentials
	env := make(map[string]string)
	for credKey, envVar := range serverDef.EnvMapping {
		if credValue, ok := creds[credKey]; ok {
			env[envVar] = credValue
		}
	}

	// Build args: base args + credential values as positional args
	args := make([]string, len(serverDef.Args))
	copy(args, serverDef.Args)
	for _, credKey := range serverDef.ArgMapping {
		if credValue, ok := creds[credKey]; ok {
			args = append(args, credValue)
		}
	}

	// Create and start backend
	stdioBackend := NewSTDIOBackend(STDIOBackendConfig{
		ID:      targetID,
		Name:    target.Name,
		Type:    target.Type,
		Command: serverDef.Command,
		Args:    args,
		Env:     env,
	})

	if err := stdioBackend.Start(ctx); err != nil {
		return fmt.Errorf("failed to start backend: %w", err)
	}

	if err := stdioBackend.Initialize(ctx); err != nil {
		stdioBackend.Close()
		return fmt.Errorf("failed to initialize backend: %w", err)
	}

	// Store instance
	instance := &BackendInstance{
		TargetID:      targetID,
		TargetName:    target.Name,
		Type:          target.Type,
		IntegrationID: target.IntegrationID,
		Backend:       stdioBackend,
		StartedAt:     time.Now(),
		LastUsed:      time.Now(),
	}

	m.mu.Lock()
	m.instances[targetID] = instance
	m.mu.Unlock()

	utils.McpLogf("[ProxyManager] Initialized target %s (type: %s)", targetID, target.Type)
	m.notifyToolsChanged()
	return nil
}

// StopTarget stops a target backend
func (m *LocalProxyManager) StopTarget(ctx context.Context, targetID string) error {
	err := m.stopTargetInternal(targetID)
	if err == nil {
		m.notifyToolsChanged()
	}
	return err
}

func (m *LocalProxyManager) stopTargetInternal(targetID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance, ok := m.instances[targetID]
	if !ok {
		return fmt.Errorf("target %s not initialized", targetID)
	}

	err := instance.Backend.Close()
	delete(m.instances, targetID)
	utils.McpLogf("[ProxyManager] Stopped target %s", targetID)
	return err
}

// ListToolsForUser returns all tools from active backends with proper prefixing
func (m *LocalProxyManager) ListToolsForUser(ctx context.Context) ([]Tool, error) {
	return m.listToolsFromBackends(ctx)
}

func (m *LocalProxyManager) listToolsFromBackends(ctx context.Context) ([]Tool, error) {
	m.mu.RLock()
	instances := make([]*BackendInstance, 0, len(m.instances))
	for _, inst := range m.instances {
		instances = append(instances, inst)
	}
	m.mu.RUnlock()

	if len(instances) == 0 {
		return []Tool{}, nil
	}

	// Group instances by type
	byType := make(map[string][]*BackendInstance)
	for _, inst := range instances {
		byType[inst.Type] = append(byType[inst.Type], inst)
	}

	var allTools []Tool

	for backendType, typeInstances := range byType {
		sort.Slice(typeInstances, func(i, j int) bool {
			return typeInstances[i].TargetID < typeInstances[j].TargetID
		})

		// Get tools from first instance
		targetID := typeInstances[0].TargetID
		instance := typeInstances[0]

		tools, err := instance.Backend.ListTools(ctx)
		if err != nil {
			utils.McpLogf("[ProxyManager] Failed to list tools from %s: %v", targetID, err)
			continue
		}

		if len(typeInstances) > 1 {
			// Multiple targets of same type: inject target enum parameter
			targetIDs := make([]string, len(typeInstances))
			for i, inst := range typeInstances {
				targetIDs[i] = inst.TargetID
			}

			for i := range tools {
				tools[i] = InjectEnumParameter(
					tools[i],
					"target",
					"Target instance to execute this operation on",
					targetIDs,
				)
				tools[i].BackendID = backendType
			}
		} else {
			// Single target: use target ID as backend ID
			for i := range tools {
				tools[i].BackendID = targetID
			}
		}

		allTools = append(allTools, tools...)
	}

	return allTools, nil
}

// routeToolCall routes a tool call to the appropriate backend
func (m *LocalProxyManager) routeToolCall(ctx context.Context, backendID string, args map[string]interface{}) (Backend, map[string]interface{}, error) {
	m.mu.RLock()
	instances := make([]*BackendInstance, 0)
	for _, inst := range m.instances {
		instances = append(instances, inst)
	}
	m.mu.RUnlock()

	// Group by type
	byType := make(map[string][]*BackendInstance)
	for _, inst := range instances {
		byType[inst.Type] = append(byType[inst.Type], inst)
	}

	// Check if backendID is a type with multiple targets
	if typeInstances, ok := byType[backendID]; ok && len(typeInstances) > 1 {
		targetIDs := make([]string, len(typeInstances))
		for i, inst := range typeInstances {
			targetIDs[i] = inst.TargetID
		}

		targetID, modifiedArgs, err := ExtractTargetFromArgs(args, "target", targetIDs)
		if err != nil {
			return nil, nil, err
		}

		backend, err := m.getBackend(ctx, targetID)
		if err != nil {
			return nil, nil, err
		}
		return backend, modifiedArgs, nil
	}

	// Direct target ID
	backend, err := m.getBackend(ctx, backendID)
	if err != nil {
		return nil, nil, err
	}
	return backend, args, nil
}

func (m *LocalProxyManager) getBackend(ctx context.Context, targetID string) (Backend, error) {
	instance := m.getInstance(targetID)
	if instance == nil {
		return nil, fmt.Errorf("target %s not initialized", targetID)
	}

	// Health check
	if err := instance.Backend.Health(ctx); err != nil {
		utils.McpLogf("[ProxyManager] Backend unhealthy for %s, respawning: %v", targetID, err)
		if err := m.InitTarget(ctx, targetID); err != nil {
			return nil, fmt.Errorf("failed to respawn backend: %w", err)
		}
		instance = m.getInstance(targetID)
		if instance == nil {
			return nil, fmt.Errorf("respawn failed")
		}
	}

	instance.UpdateLastUsed()
	return instance.Backend, nil
}

func (m *LocalProxyManager) getInstance(targetID string) *BackendInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.instances[targetID]
}

func (m *LocalProxyManager) isInitialized(targetID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if instance, ok := m.instances[targetID]; ok {
		return instance.Backend.IsReady()
	}
	return false
}

func (m *LocalProxyManager) hasBackend(backendID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check direct target ID match
	if _, ok := m.instances[backendID]; ok {
		return true
	}

	// Check if it's a type with active targets
	for _, inst := range m.instances {
		if inst.Type == backendID {
			return true
		}
	}
	return false
}

func (m *LocalProxyManager) supportedTypes() []string {
	types := make([]string, 0, len(m.mcpRegistry.Servers))
	for _, s := range m.mcpRegistry.Servers {
		types = append(types, s.ID)
	}
	return types
}

func riskLevelToString(level risk.RiskLevel) string {
	switch level {
	case risk.RiskLevelLow:
		return "low"
	case risk.RiskLevelMedium:
		return "medium"
	case risk.RiskLevelHigh:
		return "high"
	default:
		return "none"
	}
}
