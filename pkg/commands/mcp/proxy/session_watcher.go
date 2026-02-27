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
	cfg        SessionWatcherConfig
	knownReady map[string]bool // targetID -> was ready last poll
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
