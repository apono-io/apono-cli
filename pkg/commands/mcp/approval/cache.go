package approval

import (
	"context"
	"sync"

	"github.com/apono-io/apono-cli/pkg/utils"
)

// ApprovalCache wraps an Approver with in-memory caching for approved intents and patterns.
// Cache is session-scoped (lives for the MCP server process lifetime).
type ApprovalCache struct {
	delegate         Approver
	approvedIntents  map[string]bool
	approvedPatterns []string
	mu               sync.RWMutex
}

// NewApprovalCache creates a new cache wrapping the given approver.
func NewApprovalCache(delegate Approver) *ApprovalCache {
	return &ApprovalCache{
		delegate:        delegate,
		approvedIntents: make(map[string]bool),
	}
}

// RequestApproval checks the local cache first. On cache hit, returns auto-approved.
// On cache miss, delegates to the real approver and caches the result if applicable.
func (c *ApprovalCache) RequestApproval(ctx context.Context, req ApprovalRequest) (*ApprovalResult, error) {
	// Check cache
	if c.matchesCache(req) {
		utils.McpLogf("[ApprovalCache] Auto-approved from cache (tool=%s)", req.ToolName)
		return &ApprovalResult{Approved: true, Mode: ApprovalModeApproveOnce}, nil
	}

	// Cache miss — delegate to real approver
	result, err := c.delegate.RequestApproval(ctx, req)
	if err != nil {
		return nil, err
	}

	// Cache the decision if applicable
	if result.Approved {
		c.cacheResult(req, result)
	}

	return result, nil
}

// matchesCache checks if the request matches any cached intent or pattern.
func (c *ApprovalCache) matchesCache(req ApprovalRequest) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check intent cache
	if req.Intent != "" && c.approvedIntents[req.Intent] {
		utils.McpLogf("[ApprovalCache] Intent cache hit: %q", req.Intent)
		return true
	}

	// Check pattern cache
	for _, pattern := range c.approvedPatterns {
		if MatchesPattern(pattern, req.ToolName, req.Arguments) {
			utils.McpLogf("[ApprovalCache] Pattern cache hit: %q", pattern)
			return true
		}
	}

	return false
}

// cacheResult stores the approval decision based on mode.
func (c *ApprovalCache) cacheResult(req ApprovalRequest, result *ApprovalResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch result.Mode {
	case ApprovalModeApproveIntent:
		if req.Intent != "" {
			c.approvedIntents[req.Intent] = true
			utils.McpLogf("[ApprovalCache] Cached approved intent: %q", req.Intent)
		}
	case ApprovalModeApprovePattern:
		if result.Pattern != "" {
			c.approvedPatterns = append(c.approvedPatterns, result.Pattern)
			utils.McpLogf("[ApprovalCache] Cached approved pattern: %q", result.Pattern)
		}
	}
}
