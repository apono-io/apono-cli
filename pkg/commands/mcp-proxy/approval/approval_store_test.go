package approval

import (
	"testing"
	"time"

	"github.com/apono-io/apono-cli/pkg/commands/mcp-proxy/auditor"
)

func TestInMemoryApprovalStore_CreatePending(t *testing.T) {
	store := NewInMemoryApprovalStore()

	pending := PendingApproval{
		ID:        "test-123",
		Request:   auditor.RequestAudit{Method: "test"},
		CreatedAt: time.Now(),
		Done:      make(chan struct{}),
	}

	err := store.CreatePending("test-123", pending)
	if err != nil {
		t.Fatalf("Failed to create pending: %v", err)
	}

	// Try to create duplicate
	err = store.CreatePending("test-123", pending)
	if err == nil {
		t.Fatal("Expected error when creating duplicate approval")
	}
}

func TestInMemoryApprovalStore_GetPending(t *testing.T) {
	store := NewInMemoryApprovalStore()

	pending := PendingApproval{
		ID:        "test-123",
		Request:   auditor.RequestAudit{Method: "test"},
		CreatedAt: time.Now(),
		Done:      make(chan struct{}),
	}

	err := store.CreatePending("test-123", pending)
	if err != nil {
		t.Fatalf("Failed to create pending: %v", err)
	}

	retrieved, err := store.GetPending("test-123")
	if err != nil {
		t.Fatalf("Failed to get pending: %v", err)
	}

	if retrieved.ID != "test-123" {
		t.Errorf("Expected ID test-123, got %s", retrieved.ID)
	}

	// Try to get non-existent
	_, err = store.GetPending("non-existent")
	if err == nil {
		t.Fatal("Expected error when getting non-existent approval")
	}
}

func TestInMemoryApprovalStore_UpdateResponse(t *testing.T) {
	store := NewInMemoryApprovalStore()

	pending := PendingApproval{
		ID:        "test-123",
		Request:   auditor.RequestAudit{Method: "test"},
		CreatedAt: time.Now(),
		Done:      make(chan struct{}),
	}

	err := store.CreatePending("test-123", pending)
	if err != nil {
		t.Fatalf("Failed to create pending: %v", err)
	}

	response := ApprovalResponse{
		Approved:  true,
		Responder: "test-user",
		Timestamp: time.Now(),
		Comment:   "Approved for testing",
	}

	err = store.UpdateResponse("test-123", response)
	if err != nil {
		t.Fatalf("Failed to update response: %v", err)
	}

	// Verify the response was stored
	retrieved, err := store.GetPending("test-123")
	if err != nil {
		t.Fatalf("Failed to get pending: %v", err)
	}

	if retrieved.Response == nil {
		t.Fatal("Expected response to be set")
	}

	if !retrieved.Response.Approved {
		t.Error("Expected response to be approved")
	}

	if retrieved.Response.Responder != "test-user" {
		t.Errorf("Expected responder test-user, got %s", retrieved.Response.Responder)
	}

	// Verify channel is closed
	select {
	case <-retrieved.Done:
		// Channel is closed, as expected
	default:
		t.Error("Expected Done channel to be closed")
	}
}

func TestInMemoryApprovalStore_DeletePending(t *testing.T) {
	store := NewInMemoryApprovalStore()

	pending := PendingApproval{
		ID:        "test-123",
		Request:   auditor.RequestAudit{Method: "test"},
		CreatedAt: time.Now(),
		Done:      make(chan struct{}),
	}

	err := store.CreatePending("test-123", pending)
	if err != nil {
		t.Fatalf("Failed to create pending: %v", err)
	}

	err = store.DeletePending("test-123")
	if err != nil {
		t.Fatalf("Failed to delete pending: %v", err)
	}

	// Verify it's deleted
	_, err = store.GetPending("test-123")
	if err == nil {
		t.Fatal("Expected error when getting deleted approval")
	}

	// Try to delete non-existent
	err = store.DeletePending("non-existent")
	if err == nil {
		t.Fatal("Expected error when deleting non-existent approval")
	}
}
