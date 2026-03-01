package risk

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

// RiskDetector analyzes tool calls for risk
type RiskDetector interface {
	DetectRisk(toolName string, arguments map[string]interface{}) RiskDetectionResult
}

// RiskConfig holds risk detection configuration
type RiskConfig struct {
	Enabled        bool
	BlockOnRisk    bool     // If true, block risky requests without approval
	RiskyMethods   []string // Method name patterns
	RiskyKeywords  []string // Keywords in params to detect
	AllowedMethods []string // Whitelist of safe methods
}
