package auditor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/kirsle/configdir"

	"github.com/apono-io/apono-cli/pkg/config"
)

// FileAuditor implements Auditor interface with file-based storage
type FileAuditor struct {
	file *os.File
	mu   sync.Mutex // Protect concurrent writes
}

// NewFileAuditor creates a new file-based auditor
func NewFileAuditor(filePath string) (*FileAuditor, error) {
	// Use default path if not provided
	if filePath == "" {
		filePath = filepath.Join(config.DirPath, "mcp_proxy_audit.jsonl")
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := configdir.MakePath(dir); err != nil {
		return nil, fmt.Errorf("failed to create audit directory: %w", err)
	}

	// Open file in append mode (create if doesn't exist)
	file, err := os.OpenFile(
		filepath.Clean(filePath),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0600,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit file: %w", err)
	}

	return &FileAuditor{
		file: file,
	}, nil
}

// AuditRequest writes a request audit entry to the file
func (f *FileAuditor) AuditRequest(req RequestAudit) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Write as JSON line
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal audit request: %w", err)
	}

	// Append newline for JSONL format
	data = append(data, '\n')

	if _, err := f.file.Write(data); err != nil {
		return fmt.Errorf("failed to write audit log: %w", err)
	}

	return nil
}

// Close closes the audit file
func (f *FileAuditor) Close() error {
	if f.file != nil {
		return f.file.Close()
	}
	return nil
}

