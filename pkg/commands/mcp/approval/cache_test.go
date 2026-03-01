package approval

import (
	"context"
	"testing"
)

// mockApprover is a test double that records calls and returns configured results
type mockApprover struct {
	calls  []ApprovalRequest
	result *ApprovalResult
	err    error
}

func (m *mockApprover) RequestApproval(_ context.Context, req ApprovalRequest) (*ApprovalResult, error) {
	m.calls = append(m.calls, req)
	return m.result, m.err
}

func TestApprovalCache_CacheMiss_DelegatesToApprover(t *testing.T) {
	mock := &mockApprover{
		result: &ApprovalResult{Approved: true, Mode: ApprovalModeApproveOnce},
	}
	cache := NewApprovalCache(mock)

	result, err := cache.RequestApproval(context.Background(), ApprovalRequest{
		ToolName: "query",
		Intent:   "create tables",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Approved {
		t.Fatal("expected approved")
	}
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 delegate call, got %d", len(mock.calls))
	}
}

func TestApprovalCache_ApproveIntent_CachesAndAutoApproves(t *testing.T) {
	mock := &mockApprover{
		result: &ApprovalResult{Approved: true, Mode: ApprovalModeApproveIntent},
	}
	cache := NewApprovalCache(mock)
	ctx := context.Background()

	// First call: delegates to approver
	_, err := cache.RequestApproval(ctx, ApprovalRequest{
		ToolName: "query",
		Intent:   "create tables",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second call with same intent: should auto-approve without delegating
	result, err := cache.RequestApproval(ctx, ApprovalRequest{
		ToolName: "query",
		Intent:   "create tables",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Approved {
		t.Fatal("expected auto-approved")
	}
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 delegate call (cached), got %d", len(mock.calls))
	}
}

func TestApprovalCache_ApprovePattern_CachesAndAutoApproves(t *testing.T) {
	mock := &mockApprover{
		result: &ApprovalResult{Approved: true, Mode: ApprovalModeApprovePattern, Pattern: "query:CREATE*"},
	}
	cache := NewApprovalCache(mock)
	ctx := context.Background()

	// First call: delegates to approver
	_, err := cache.RequestApproval(ctx, ApprovalRequest{
		ToolName:  "query",
		Arguments: map[string]interface{}{"sql": "CREATE TABLE users (id INT)"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second call with different SQL but same prefix: should auto-approve
	result, err := cache.RequestApproval(ctx, ApprovalRequest{
		ToolName:  "query",
		Arguments: map[string]interface{}{"sql": "CREATE TABLE orders (id INT)"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Approved {
		t.Fatal("expected auto-approved via pattern")
	}
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 delegate call (cached), got %d", len(mock.calls))
	}
}

func TestApprovalCache_ApprovePattern_NoMatchDelegates(t *testing.T) {
	mock := &mockApprover{
		result: &ApprovalResult{Approved: true, Mode: ApprovalModeApprovePattern, Pattern: "query:CREATE*"},
	}
	cache := NewApprovalCache(mock)
	ctx := context.Background()

	// First call: CREATE TABLE
	_, _ = cache.RequestApproval(ctx, ApprovalRequest{
		ToolName:  "query",
		Arguments: map[string]interface{}{"sql": "CREATE TABLE users (id INT)"},
	})

	// Second call: DROP TABLE — should NOT match CREATE* pattern, delegates to approver
	mock.result = &ApprovalResult{Approved: false, Mode: ApprovalModeDeny}
	_, _ = cache.RequestApproval(ctx, ApprovalRequest{
		ToolName:  "query",
		Arguments: map[string]interface{}{"sql": "DROP TABLE users"},
	})

	if len(mock.calls) != 2 {
		t.Fatalf("expected 2 delegate calls (pattern mismatch), got %d", len(mock.calls))
	}
}

func TestApprovalCache_DenyDoesNotCache(t *testing.T) {
	mock := &mockApprover{
		result: &ApprovalResult{Approved: false, Mode: ApprovalModeDeny},
	}
	cache := NewApprovalCache(mock)
	ctx := context.Background()

	// First call: denied
	_, _ = cache.RequestApproval(ctx, ApprovalRequest{
		ToolName: "query",
		Intent:   "drop tables",
	})

	// Second call with same intent: should still delegate (deny is not cached)
	mock.result = &ApprovalResult{Approved: true, Mode: ApprovalModeApproveOnce}
	_, _ = cache.RequestApproval(ctx, ApprovalRequest{
		ToolName: "query",
		Intent:   "drop tables",
	})

	if len(mock.calls) != 2 {
		t.Fatalf("expected 2 delegate calls (deny not cached), got %d", len(mock.calls))
	}
}

func TestApprovalCache_EmptyIntent_NoIntentCacheHit(t *testing.T) {
	mock := &mockApprover{
		result: &ApprovalResult{Approved: true, Mode: ApprovalModeApproveIntent},
	}
	cache := NewApprovalCache(mock)
	ctx := context.Background()

	// Call with empty intent
	_, _ = cache.RequestApproval(ctx, ApprovalRequest{
		ToolName: "query",
		Intent:   "",
	})

	// Second call with empty intent: should still delegate (empty intent not cached)
	_, _ = cache.RequestApproval(ctx, ApprovalRequest{
		ToolName: "query",
		Intent:   "",
	})

	if len(mock.calls) != 2 {
		t.Fatalf("expected 2 delegate calls (empty intent), got %d", len(mock.calls))
	}
}
