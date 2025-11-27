package approval

import (
	"context"
	"fmt"
	"time"

	"github.com/apono-io/apono-cli/pkg/commands/mcp-proxy/auditor"
	"github.com/google/uuid"
)

// SlackNotifier is the interface for sending approval requests to Slack
type SlackNotifier interface {
	SendApprovalRequest(ctx context.Context, req ApprovalRequest) (string, error)
}

// ApprovalManager orchestrates the approval workflow
type ApprovalManager struct {
	store    *InMemoryApprovalStore
	notifier SlackNotifier
	timeout  time.Duration
}

// NewApprovalManager creates a new approval manager
func NewApprovalManager(store *InMemoryApprovalStore, notifier SlackNotifier, timeout time.Duration) *ApprovalManager {
	return &ApprovalManager{
		store:    store,
		notifier: notifier,
		timeout:  timeout,
	}
}

// RequestApproval sends an approval request to Slack and waits for a response
// Returns true if approved, false if denied or timed out
func (am *ApprovalManager) RequestApproval(ctx context.Context, req auditor.RequestAudit) (bool, *auditor.ApprovalResponse, error) {
	// Generate unique approval ID
	approvalID := uuid.New().String()

	// Create approval request
	approvalReq := ApprovalRequest{
		ID:         approvalID,
		Timestamp:  time.Now(),
		Method:     req.Method,
		ClientName: req.ClientName,
		Risk:       *req.Risk,
		Params:     req.Params,
	}

	// Create pending approval with done channel
	pending := PendingApproval{
		ID:        approvalID,
		Request:   req,
		CreatedAt: time.Now(),
		Response:  nil,
		Done:      make(chan struct{}),
	}

	// Store pending approval
	if err := am.store.CreatePending(approvalID, pending); err != nil {
		return false, nil, fmt.Errorf("failed to create pending approval: %w", err)
	}

	// Cleanup after we're done
	defer am.store.DeletePending(approvalID)

	// Send Slack notification
	if _, err := am.notifier.SendApprovalRequest(ctx, approvalReq); err != nil {
		return false, nil, fmt.Errorf("failed to send Slack notification: %w", err)
	}

	// Wait for response or timeout
	select {
	case <-pending.Done:
		// Response received
		finalPending, err := am.store.GetPending(approvalID)
		if err != nil {
			return false, nil, fmt.Errorf("failed to get approval response: %w", err)
		}

		if finalPending.Response == nil {
			return false, nil, fmt.Errorf("approval response is nil")
		}

		// Convert to auditor.ApprovalResponse
		auditorResp := &auditor.ApprovalResponse{
			Approved:  finalPending.Response.Approved,
			Responder: finalPending.Response.Responder,
			Timestamp: finalPending.Response.Timestamp,
			Comment:   finalPending.Response.Comment,
		}

		return finalPending.Response.Approved, auditorResp, nil

	case <-time.After(am.timeout):
		// Timeout - auto-deny
		return false, nil, fmt.Errorf("approval request timed out after %v", am.timeout)

	case <-ctx.Done():
		// Context cancelled
		return false, nil, fmt.Errorf("approval request cancelled: %w", ctx.Err())
	}
}
