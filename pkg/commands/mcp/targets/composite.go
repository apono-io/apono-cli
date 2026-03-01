package targets

import (
	"context"
	"fmt"

	"github.com/apono-io/apono-cli/pkg/utils"
)

// CompositeTargetSource merges multiple TargetSources
// File targets take priority over session targets on ID conflict
type CompositeTargetSource struct {
	sources []TargetSource
}

// NewCompositeTargetSource creates a composite source from the given sources
// Sources are listed in priority order (first source wins on conflict)
func NewCompositeTargetSource(sources ...TargetSource) *CompositeTargetSource {
	return &CompositeTargetSource{
		sources: sources,
	}
}

// ListTargets merges targets from all sources
// File targets override session targets on ID conflict
func (c *CompositeTargetSource) ListTargets(ctx context.Context) ([]TargetInfo, error) {
	seen := make(map[string]bool)
	merged := make([]TargetInfo, 0)

	for _, source := range c.sources {
		targets, err := source.ListTargets(ctx)
		if err != nil {
			utils.McpLogf("[CompositeSource] Error listing targets from source: %v", err)
			continue
		}

		for _, t := range targets {
			if !seen[t.ID] {
				seen[t.ID] = true
				merged = append(merged, t)
			}
		}
	}

	return merged, nil
}

// GetTarget returns a target definition, checking sources in priority order
func (c *CompositeTargetSource) GetTarget(ctx context.Context, targetID string) (*TargetDefinition, error) {
	for _, source := range c.sources {
		target, err := source.GetTarget(ctx, targetID)
		if err == nil {
			return target, nil
		}
	}

	return nil, fmt.Errorf("target %q not found in any source", targetID)
}

// EnsureAccess delegates to the source that owns the target.
// Uses ListTargets (not GetTarget) to find ownership, avoiding credential consumption.
func (c *CompositeTargetSource) EnsureAccess(ctx context.Context, targetID string) error {
	for _, source := range c.sources {
		targets, err := source.ListTargets(ctx)
		if err != nil {
			continue
		}
		for _, t := range targets {
			if t.ID == targetID {
				return source.EnsureAccess(ctx, targetID)
			}
		}
	}

	return fmt.Errorf("target %q not found in any source", targetID)
}
