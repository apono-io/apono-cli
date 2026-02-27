package proxy

import (
	"context"
	"fmt"
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
				Command:          "echo",
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

func TestSessionWatcher_DetectsExpiredSession(t *testing.T) {
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
				Command:          "echo",
			},
		},
	}

	expiredCalled := make(chan string, 1)
	pollCount := 0

	watcher := NewSessionWatcher(SessionWatcherConfig{
		TargetSource: source,
		MCPRegistry:  reg,
		PollInterval: 50 * time.Millisecond,
		OnNewSession: func(targetID string, serverDef registry.MCPServerDefinition, target *targets.TargetDefinition) {
			// After first detection, remove the target to simulate expiry
			pollCount++
			if pollCount >= 1 {
				source.targets = []targets.TargetInfo{} // empty = expired
			}
		},
		OnExpiredSession: func(targetID string) {
			expiredCalled <- targetID
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	go watcher.Start(ctx)

	select {
	case id := <-expiredCalled:
		if id != "prod-pg" {
			t.Errorf("expected prod-pg, got %s", id)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for expiry detection")
	}
}
