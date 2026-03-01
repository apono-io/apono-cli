package approval

import "context"

// ApprovalRequest represents a request for human approval of a risky operation
type ApprovalRequest struct {
	ToolName      string                 `json:"tool_name"`
	Arguments     map[string]interface{} `json:"arguments"`
	Reason        string                 `json:"reason"`         // why approval is needed
	RiskLevel     string                 `json:"risk_level"`     // "low", "medium", "high"
	MatchedRule   string                 `json:"matched_rule"`   // which risk rule was matched (e.g., "keyword:DROP TABLE")
	TargetID      string                 `json:"target_id"`      // which backend target
	IntegrationID string                 `json:"integration_id"` // Apono integration ID for creating access request
}

// Approver requests approval for risky operations
type Approver interface {
	// RequestApproval submits an approval request and blocks until approved/denied or timeout
	RequestApproval(ctx context.Context, req ApprovalRequest) (approved bool, err error)
}
