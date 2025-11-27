package auditor

import (
	"encoding/json"
	"time"
)

// Auditor is the interface for auditing incoming MCP requests
type Auditor interface {
	AuditRequest(req RequestAudit) error
	Close() error
}

// RequestAudit represents an audited request with extracted content
type RequestAudit struct {
	Timestamp  time.Time              `json:"timestamp"`
	Method     string                 `json:"method"`
	ClientName string                 `json:"client_name,omitempty"`
	Params     map[string]interface{} `json:"params,omitempty"`
	Mode       string                 `json:"mode"` // "http" or "stdio"
	RequestID  interface{}            `json:"request_id,omitempty"`
	Risk       *RiskDetectionResult   `json:"risk,omitempty"`
	Blocked    bool                   `json:"blocked,omitempty"`

	// Approval tracking fields
	ApprovalRequested bool      `json:"approval_requested,omitempty"`
	Approved          bool      `json:"approved,omitempty"`
	ApprovedBy        string    `json:"approved_by,omitempty"`
	ApprovedAt        time.Time `json:"approved_at,omitempty"`
}

// AuditorConfig holds configuration for creating an auditor
type AuditorConfig struct {
	Type     string // "file" or "database"
	FilePath string // for file type
	// Future: DBConfig *DatabaseConfig // for database type
}

// NewAuditor creates an auditor based on configuration
// For MVP, returns file auditor. Later can return DB auditor based on config
func NewAuditor(config AuditorConfig) (Auditor, error) {
	switch config.Type {
	case "file":
		return NewFileAuditor(config.FilePath)
	// Future: case "database": return NewDBauditor(config.DBConfig)
	default:
		return NewFileAuditor(config.FilePath)
	}
}

// ExtractRequestContent parses JSON-RPC request and extracts meaningful content
func ExtractRequestContent(rawRequest string, clientName string, mode string) (*RequestAudit, error) {
	var jsonRPC struct {
		Method string                 `json:"method"`
		Params map[string]interface{} `json:"params"`
		ID     interface{}            `json:"id"`
	}

	if err := json.Unmarshal([]byte(rawRequest), &jsonRPC); err != nil {
		return nil, err
	}

	return &RequestAudit{
		Timestamp:  time.Now(),
		Method:     jsonRPC.Method,
		ClientName: clientName,
		Params:     jsonRPC.Params,
		Mode:       mode,
		RequestID:  jsonRPC.ID,
	}, nil
}
