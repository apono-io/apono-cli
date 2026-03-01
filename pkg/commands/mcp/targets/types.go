package targets

import "context"

// TargetStatus represents the access status of a target
type TargetStatus string

const (
	TargetStatusReady       TargetStatus = "ready"        // has active session with credentials or from targets.yaml
	TargetStatusNeedsAccess TargetStatus = "needs_access" // integration exists but no active session
	TargetStatusPending     TargetStatus = "pending"      // access request in progress
)

// TargetDefinition represents a target configuration with credentials
type TargetDefinition struct {
	ID            string            `yaml:"id" json:"id"`
	Name          string            `yaml:"name" json:"name"`
	Type          string            `yaml:"type" json:"type"`               // references MCPServerDefinition.ID (e.g., "postgres")
	Credentials   map[string]string `yaml:"credentials" json:"credentials"` // key-value credential pairs
	IntegrationID string            `yaml:"-" json:"-"`                     // Apono integration ID (set by session provider, not from file)
}

// TargetInfo is the user-facing representation returned by list_targets
type TargetInfo struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Type        string       `json:"type"`
	Status      TargetStatus `json:"status"`
	Initialized bool         `json:"initialized"` // whether a backend is currently running
}

// TargetSource provides target definitions with status
type TargetSource interface {
	// ListTargets returns all available targets with their current status
	ListTargets(ctx context.Context) ([]TargetInfo, error)

	// GetTarget returns a specific target's definition with credentials (if available)
	GetTarget(ctx context.Context, targetID string) (*TargetDefinition, error)

	// EnsureAccess ensures the target is accessible, potentially requesting access and blocking until approved
	EnsureAccess(ctx context.Context, targetID string) error
}

// TargetsFile represents the targets.yaml file structure
type TargetsFile struct {
	Targets []TargetDefinition `yaml:"targets"`
}
