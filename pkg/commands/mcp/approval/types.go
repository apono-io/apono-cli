package approval

import "context"

// ApprovalMode represents the type of approval decision
type ApprovalMode string

const (
	ApprovalModeApproveOnce    ApprovalMode = "approve_once"
	ApprovalModeDeny           ApprovalMode = "deny"
	ApprovalModeApproveIntent  ApprovalMode = "approve_intent"
	ApprovalModeApprovePattern ApprovalMode = "approve_pattern"
)

// ApprovalResult represents the outcome of an approval request
type ApprovalResult struct {
	Approved bool
	Mode     ApprovalMode
	Pattern  string // set when Mode == ApprovalModeApprovePattern
}

// ApprovalRequest represents a request for human approval of a risky operation
type ApprovalRequest struct {
	ToolName         string                 `json:"tool_name"`
	Arguments        map[string]interface{} `json:"arguments"`
	Reason           string                 `json:"reason"`
	RiskLevel        string                 `json:"risk_level"`
	MatchedRule      string                 `json:"matched_rule"`
	TargetID         string                 `json:"target_id"`
	IntegrationID    string                 `json:"integration_id"`
	Intent           string                 `json:"intent,omitempty"`
	SuggestedPattern string                 `json:"suggested_pattern,omitempty"`
}

// Approver requests approval for risky operations
type Approver interface {
	// RequestApproval submits an approval request and blocks until a decision is made or timeout
	RequestApproval(ctx context.Context, req ApprovalRequest) (*ApprovalResult, error)
}
