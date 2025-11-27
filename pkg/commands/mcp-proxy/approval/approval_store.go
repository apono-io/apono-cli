package approval

import (
	"fmt"
	"sync"
)

// ApprovalStore manages pending approval requests
type ApprovalStore interface {
	CreatePending(approvalID string, request PendingApproval) error
	GetPending(approvalID string) (*PendingApproval, error)
	UpdateResponse(approvalID string, response ApprovalResponse) error
	DeletePending(approvalID string) error
}

// InMemoryApprovalStore is a thread-safe in-memory implementation of ApprovalStore
type InMemoryApprovalStore struct {
	mu        sync.RWMutex
	approvals map[string]*PendingApproval
}

// NewInMemoryApprovalStore creates a new in-memory approval store
func NewInMemoryApprovalStore() *InMemoryApprovalStore {
	return &InMemoryApprovalStore{
		approvals: make(map[string]*PendingApproval),
	}
}

// CreatePending adds a new pending approval to the store
func (s *InMemoryApprovalStore) CreatePending(approvalID string, request PendingApproval) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.approvals[approvalID]; exists {
		return fmt.Errorf("approval with ID %s already exists", approvalID)
	}

	s.approvals[approvalID] = &request
	return nil
}

// GetPending retrieves a pending approval by ID
func (s *InMemoryApprovalStore) GetPending(approvalID string) (*PendingApproval, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	approval, exists := s.approvals[approvalID]
	if !exists {
		return nil, fmt.Errorf("approval with ID %s not found", approvalID)
	}

	return approval, nil
}

// UpdateResponse updates a pending approval with a response and signals completion
func (s *InMemoryApprovalStore) UpdateResponse(approvalID string, response ApprovalResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	approval, exists := s.approvals[approvalID]
	if !exists {
		return fmt.Errorf("approval with ID %s not found", approvalID)
	}

	// Only update if not already responded
	if approval.Response != nil {
		return fmt.Errorf("approval with ID %s already has a response", approvalID)
	}

	approval.Response = &response

	// Signal waiting goroutine
	close(approval.Done)

	return nil
}

// DeletePending removes a pending approval from the store
func (s *InMemoryApprovalStore) DeletePending(approvalID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.approvals[approvalID]; !exists {
		return fmt.Errorf("approval with ID %s not found", approvalID)
	}

	delete(s.approvals, approvalID)
	return nil
}
