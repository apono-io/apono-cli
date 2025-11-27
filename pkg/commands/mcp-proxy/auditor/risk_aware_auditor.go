package auditor

import "fmt"

// RiskAwareAuditor wraps an auditor with risk detection capabilities
type RiskAwareAuditor struct {
	auditor      Auditor
	riskDetector RiskDetector
	blockOnRisk  bool
}

// NewRiskAwareAuditor creates a new risk-aware auditor
func NewRiskAwareAuditor(auditor Auditor, detector RiskDetector, blockOnRisk bool) *RiskAwareAuditor {
	return &RiskAwareAuditor{
		auditor:      auditor,
		riskDetector: detector,
		blockOnRisk:  blockOnRisk,
	}
}

// AuditRequest audits a request and checks for risks
func (r *RiskAwareAuditor) AuditRequest(req RequestAudit) error {
	// Check for risks first
	riskResult := r.riskDetector.DetectRisk(req)

	// Add risk information to the audit record
	req.Risk = &riskResult

	if riskResult.IsRisky {
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

