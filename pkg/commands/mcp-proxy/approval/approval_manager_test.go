package approval

import (
	"context"
	"testing"
	"time"

	"github.com/apono-io/apono-cli/pkg/commands/mcp-proxy/auditor"
)

// MockSlackNotifier is a mock implementation of SlackNotifier for testing
type MockSlackNotifier struct {
	sendCalled bool
	sendError  error
}

func (m *MockSlackNotifier) SendApprovalRequest(ctx context.Context, req ApprovalRequest) (string, error) {
	m.sendCalled = true
	if m.sendError != nil {
		return "", m.sendError
	}
	return "mock-message-id", nil
}

func TestApprovalManager_RequestApproval_Timeout(t *testing.T) {
	store := NewInMemoryApprovalStore()
	notifier := &MockSlackNotifier{}
	manager := NewApprovalManager(store, notifier, 100*time.Millisecond)

	req := auditor.RequestAudit{
		Method:     "delete",
		ClientName: "test-client",
		Risk: &auditor.RiskDetectionResult{
			IsRisky: true,
			Level:   3,
			Reason:  "Dangerous operation",
		},
	}

	ctx := context.Background()
	approved, resp, err := manager.RequestApproval(ctx, req)

	if err == nil {
		t.Fatal("Expected error on timeout")
	}

	if approved {
		t.Error("Expected approval to be false on timeout")
	}

	if resp != nil {
		t.Error("Expected response to be nil on timeout")
	}

	if !notifier.sendCalled {
		t.Error("Expected notifier to be called")
	}
}

func TestApprovalManager_RequestApproval_Approved(t *testing.T) {
	store := NewInMemoryApprovalStore()
	notifier := &MockSlackNotifier{}
	manager := NewApprovalManager(store, notifier, 5*time.Second)

	req := auditor.RequestAudit{
		Method:     "delete",
		ClientName: "test-client",
		Risk: &auditor.RiskDetectionResult{
			IsRisky: true,
			Level:   3,
			Reason:  "Dangerous operation",
		},
	}

	// Simulate approval in background
	go func() {
		time.Sleep(100 * time.Millisecond)

		// Find the pending approval
		store.mu.RLock()
		var approvalID string
		for id := range store.approvals {
			approvalID = id
			break
		}
		store.mu.RUnlock()

		if approvalID != "" {
			response := ApprovalResponse{
				Approved:  true,
				Responder: "test-approver",
				Timestamp: time.Now(),
			}
			store.UpdateResponse(approvalID, response)
		}
	}()

	ctx := context.Background()
	approved, resp, err := manager.RequestApproval(ctx, req)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !approved {
		t.Error("Expected approval to be true")
	}

	if resp == nil {
		t.Fatal("Expected response to be non-nil")
	}

	if resp.Responder != "test-approver" {
		t.Errorf("Expected responder test-approver, got %s", resp.Responder)
	}

	if !notifier.sendCalled {
		t.Error("Expected notifier to be called")
	}
}

func TestApprovalManager_RequestApproval_Denied(t *testing.T) {
	store := NewInMemoryApprovalStore()
	notifier := &MockSlackNotifier{}
	manager := NewApprovalManager(store, notifier, 5*time.Second)

	req := auditor.RequestAudit{
		Method:     "delete",
		ClientName: "test-client",
		Risk: &auditor.RiskDetectionResult{
			IsRisky: true,
			Level:   3,
			Reason:  "Dangerous operation",
		},
	}

	// Simulate denial in background
	go func() {
		time.Sleep(100 * time.Millisecond)

		// Find the pending approval
		store.mu.RLock()
		var approvalID string
		for id := range store.approvals {
			approvalID = id
			break
		}
		store.mu.RUnlock()

		if approvalID != "" {
			response := ApprovalResponse{
				Approved:  false,
				Responder: "test-denier",
				Timestamp: time.Now(),
			}
			store.UpdateResponse(approvalID, response)
		}
	}()

	ctx := context.Background()
	approved, resp, err := manager.RequestApproval(ctx, req)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if approved {
		t.Error("Expected approval to be false")
	}

	if resp == nil {
		t.Fatal("Expected response to be non-nil")
	}

	if resp.Responder != "test-denier" {
		t.Errorf("Expected responder test-denier, got %s", resp.Responder)
	}
}
