package risk

import (
	"encoding/json"
	"strings"
)

// PatternRiskDetector implements pattern-based risk detection
type PatternRiskDetector struct {
	config RiskConfig
}

// DefaultRiskConfig returns sensible defaults
func DefaultRiskConfig() RiskConfig {
	return RiskConfig{
		Enabled:     true,
		BlockOnRisk: true,
		RiskyMethods: []string{
			"delete",
			"remove",
			"destroy",
			"drop",
			"truncate",
			"execute",
			"exec",
			"run",
		},
		RiskyKeywords: []string{
			// SQL dangerous operations
			"DROP TABLE",
			"DROP DATABASE",
			"DELETE FROM",
			"TRUNCATE",
			"ALTER TABLE",
			"DROP INDEX",
			"DROP SCHEMA",
			// File operations
			"rm -rf",
			"rm -r",
			"unlink",
			"format",
			// Shell operations
			"exec(",
			"system(",
			"shell_exec",
		},
		AllowedMethods: []string{
			"initialize",
			"list",
			"read",
			"get",
			"search",
			"resources/list",
			"resources/read",
			"tools/list",
			"prompts/list",
		},
	}
}

// NewPatternRiskDetector creates a new pattern-based risk detector
func NewPatternRiskDetector(config RiskConfig) *PatternRiskDetector {
	return &PatternRiskDetector{config: config}
}

// DetectRisk analyzes a tool call for risky operations
func (d *PatternRiskDetector) DetectRisk(toolName string, arguments map[string]interface{}) RiskDetectionResult {
	if !d.config.Enabled {
		return RiskDetectionResult{IsRisky: false, Level: RiskLevelNone}
	}

	method := strings.ToLower(toolName)

	// Check whitelist first
	for _, allowed := range d.config.AllowedMethods {
		if strings.Contains(method, strings.ToLower(allowed)) {
			// Still check params for dangerous content
			if result := d.checkParams(arguments); result.IsRisky {
				return result
			}
			return RiskDetectionResult{IsRisky: false, Level: RiskLevelNone}
		}
	}

	// Check risky method patterns
	for _, pattern := range d.config.RiskyMethods {
		if strings.Contains(method, strings.ToLower(pattern)) {
			return RiskDetectionResult{
				IsRisky:     true,
				Level:       RiskLevelHigh,
				Reason:      "Method contains risky operation pattern",
				MatchedRule: "method:" + pattern,
			}
		}
	}

	// Check params for dangerous content
	if result := d.checkParams(arguments); result.IsRisky {
		return result
	}

	return RiskDetectionResult{IsRisky: false, Level: RiskLevelNone}
}

// checkParams inspects parameters for dangerous keywords
func (d *PatternRiskDetector) checkParams(params map[string]interface{}) RiskDetectionResult {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return RiskDetectionResult{IsRisky: false, Level: RiskLevelNone}
	}

	paramsStr := strings.ToUpper(string(paramsJSON))

	for _, keyword := range d.config.RiskyKeywords {
		if strings.Contains(paramsStr, strings.ToUpper(keyword)) {
			return RiskDetectionResult{
				IsRisky:     true,
				Level:       RiskLevelHigh,
				Reason:      "Detected dangerous operation in parameters",
				MatchedRule: "keyword:" + keyword,
			}
		}
	}

	return RiskDetectionResult{IsRisky: false, Level: RiskLevelNone}
}
