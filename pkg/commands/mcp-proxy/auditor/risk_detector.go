package auditor

// RiskLevel represents the severity of detected risk
type RiskLevel int

const (
	RiskLevelNone RiskLevel = iota
	RiskLevelLow
	RiskLevelMedium
	RiskLevelHigh
)

// RiskDetectionResult contains risk analysis results
type RiskDetectionResult struct {
	IsRisky     bool      `json:"is_risky"`
	Level       RiskLevel `json:"level"`
	Reason      string    `json:"reason,omitempty"`
	MatchedRule string    `json:"matched_rule,omitempty"`
}

// RiskDetector analyzes requests for risky operations
type RiskDetector interface {
	DetectRisk(req RequestAudit) RiskDetectionResult
}

