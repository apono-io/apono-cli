package auditor

// NoopAuditor implements Auditor interface but does nothing
// Useful for testing and when auditing needs to be disabled
type NoopAuditor struct{}

// NewNoopAuditor creates a new no-op auditor
func NewNoopAuditor() *NoopAuditor {
	return &NoopAuditor{}
}

// AuditRequest does nothing
func (n *NoopAuditor) AuditRequest(req RequestAudit) error {
	return nil
}

// Close does nothing
func (n *NoopAuditor) Close() error {
	return nil
}

