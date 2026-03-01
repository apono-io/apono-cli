package proxy

import (
	"time"

	"github.com/apono-io/apono-cli/pkg/utils"
)

// StartCleanupRoutine starts a background goroutine that cleans up idle backends
func (m *LocalProxyManager) StartCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				m.cleanupIdleBackends()
			case <-m.done:
				return
			}
		}
	}()
}

// cleanupIdleBackends removes backends that have been idle for longer than cleanupTimeout
func (m *LocalProxyManager) cleanupIdleBackends() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for targetID, instance := range m.instances {
		if now.Sub(instance.GetLastUsed()) > m.cleanupTimeout {
			utils.McpLogf("[Cleanup] Removing idle target %s (idle for %v)",
				targetID, now.Sub(instance.GetLastUsed()))

			if err := instance.Backend.Close(); err != nil {
				utils.McpLogf("[Cleanup] Error closing idle backend %s: %v", targetID, err)
			}

			delete(m.instances, targetID)
		}
	}
}
