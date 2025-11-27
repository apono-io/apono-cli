package approval

import (
	"time"

	"github.com/apono-io/apono-cli/pkg/commands/mcp-proxy/auditor"
)

// ApprovalRequest represents a request for approval of a risky operation
type ApprovalRequest struct {
	ID         string                     `json:"id"`
	Timestamp  time.Time                  `json:"timestamp"`
	Method     string                     `json:"method"`
	ClientName string                     `json:"client_name,omitempty"`
	Risk       auditor.RiskDetectionResult `json:"risk"`
	Params     map[string]interface{}     `json:"params,omitempty"`
}

// ApprovalResponse represents the response from an approver
type ApprovalResponse struct {
	Approved  bool      `json:"approved"`
	Responder string    `json:"responder"`
	Timestamp time.Time `json:"timestamp"`
	Comment   string    `json:"comment,omitempty"`
}

// PendingApproval represents an approval request waiting for a response
type PendingApproval struct {
	ID        string
	Request   auditor.RequestAudit
	CreatedAt time.Time
	Response  *ApprovalResponse
	Done      chan struct{} // Signal when response received
}

