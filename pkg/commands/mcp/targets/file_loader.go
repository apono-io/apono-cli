package targets

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/apono-io/apono-cli/pkg/utils"
	"gopkg.in/yaml.v3"
)

// FileTargetLoader loads targets from a YAML file
// All file-based targets are always "ready" (they have credentials embedded)
type FileTargetLoader struct {
	filePath string
}

// NewFileTargetLoader creates a new file-based target loader
func NewFileTargetLoader(filePath string) *FileTargetLoader {
	return &FileTargetLoader{
		filePath: filePath,
	}
}

// ListTargets returns all targets from the file, all with status "ready"
func (l *FileTargetLoader) ListTargets(ctx context.Context) ([]TargetInfo, error) {
	targets, err := l.loadTargets()
	if err != nil {
		// If file doesn't exist, return empty list (not an error)
		if os.IsNotExist(err) {
			return []TargetInfo{}, nil
		}
		return nil, err
	}

	result := make([]TargetInfo, 0, len(targets))
	for _, t := range targets {
		result = append(result, TargetInfo{
			ID:     t.ID,
			Name:   t.Name,
			Type:   t.Type,
			Status: TargetStatusReady,
		})
	}

	return result, nil
}

// GetTarget returns a specific target's definition
func (l *FileTargetLoader) GetTarget(ctx context.Context, targetID string) (*TargetDefinition, error) {
	targets, err := l.loadTargets()
	if err != nil {
		return nil, err
	}

	for _, t := range targets {
		if t.ID == targetID {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("target %q not found in targets file", targetID)
}

// EnsureAccess is a no-op for file targets (they always have credentials)
func (l *FileTargetLoader) EnsureAccess(ctx context.Context, targetID string) error {
	return nil
}

// AddTarget adds or updates a target in the targets file
func (l *FileTargetLoader) AddTarget(target TargetDefinition) error {
	var targetsFile TargetsFile

	data, err := os.ReadFile(l.filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read targets file: %w", err)
		}
		// File doesn't exist — we'll create it
	} else {
		if err := yaml.Unmarshal(data, &targetsFile); err != nil {
			return fmt.Errorf("failed to parse targets file: %w", err)
		}
	}

	// Update existing target or append new one
	found := false
	for i, t := range targetsFile.Targets {
		if t.ID == target.ID {
			targetsFile.Targets[i] = target
			found = true
			break
		}
	}
	if !found {
		targetsFile.Targets = append(targetsFile.Targets, target)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(l.filePath), 0o755); err != nil {
		return fmt.Errorf("failed to create targets directory: %w", err)
	}

	out, err := yaml.Marshal(&targetsFile)
	if err != nil {
		return fmt.Errorf("failed to marshal targets file: %w", err)
	}

	if err := os.WriteFile(l.filePath, out, 0o644); err != nil {
		return fmt.Errorf("failed to write targets file: %w", err)
	}

	utils.McpLogf("Added/updated target %s in %s", target.ID, l.filePath)
	return nil
}

// loadTargets reads and parses the targets file fresh each time
func (l *FileTargetLoader) loadTargets() ([]TargetDefinition, error) {
	data, err := os.ReadFile(l.filePath)
	if err != nil {
		return nil, err
	}

	var targetsFile TargetsFile
	if err := yaml.Unmarshal(data, &targetsFile); err != nil {
		return nil, fmt.Errorf("failed to parse targets file: %w", err)
	}

	utils.McpLogf("Loaded %d targets from %s", len(targetsFile.Targets), l.filePath)
	return targetsFile.Targets, nil
}
