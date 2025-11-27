package auditor

import (
	"context"
	"fmt"
	"time"
)

// ApprovalManager is the interface for requesting approval of risky operations
type ApprovalManager interface {
	RequestApproval(ctx context.Context, req RequestAudit) (bool, *ApprovalResponse, error)
}

// ApprovalResponse represents the response from an approver
type ApprovalResponse struct {
	Approved  bool
	Responder string
	Timestamp time.Time
	Comment   string
}

// RiskAwareAuditor wraps an auditor with risk detection capabilities
type RiskAwareAuditor struct {
	auditor         Auditor
	riskDetector    RiskDetector
	blockOnRisk     bool
	approvalManager ApprovalManager
	requireApproval bool
}

// NewRiskAwareAuditor creates a new risk-aware auditor
func NewRiskAwareAuditor(auditor Auditor, detector RiskDetector, blockOnRisk bool, approvalManager ApprovalManager, requireApproval bool) *RiskAwareAuditor {
	return &RiskAwareAuditor{
		auditor:         auditor,
		riskDetector:    detector,
		blockOnRisk:     blockOnRisk,
		approvalManager: approvalManager,
		requireApproval: requireApproval,
	}
}

// AuditRequest audits a request and checks for risks
func (r *RiskAwareAuditor) AuditRequest(req RequestAudit) error {
	// Check for risks first
	riskResult := r.riskDetector.DetectRisk(req)

	// Add risk information to the audit record
	req.Risk = &riskResult

	if riskResult.IsRisky {
		// Check if approval workflow is enabled
		if r.requireApproval && r.approvalManager != nil {
			req.ApprovalRequested = true

			// Request approval via Slack
			approved, approvalResp, err := r.approvalManager.RequestApproval(context.Background(), req)

			// Update audit record with approval information
			req.Approved = approved
			req.Blocked = !approved

			if approvalResp != nil {
				req.ApprovedBy = approvalResp.Responder
				req.ApprovedAt = approvalResp.Timestamp
			}

			// Log the request with approval outcome
			auditErr := r.auditor.AuditRequest(req)
			if auditErr != nil {
				// Log error but continue with the decision
			}

			if err != nil || !approved {
				if err != nil {
					return fmt.Errorf("BLOCKED: Approval request failed - %w", err)
				}
				return fmt.Errorf("BLOCKED: Approval denied by %s", req.ApprovedBy)
			}

			// Approved - continue execution
			return nil
		}

		// Fallback to immediate blocking if approval not enabled
		req.Blocked = r.blockOnRisk

		// Always log risky requests (even if blocked)
		auditErr := r.auditor.AuditRequest(req)
		if auditErr != nil {
			// Log but continue to block if needed
		}

		if r.blockOnRisk {
			return fmt.Errorf("BLOCKED: Risky operation detected - %s (rule: %s)",
				riskResult.Reason, riskResult.MatchedRule)
		}
	}

	// Normal audit for non-risky requests
	return r.auditor.AuditRequest(req)
}

// Close closes the underlying auditor
func (r *RiskAwareAuditor) Close() error {
	return r.auditor.Close()
}
