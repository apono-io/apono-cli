package approval

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/apono-io/apono-cli/pkg/utils"
)

const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusDenied   = "denied"

	defaultPollInterval = 3 * time.Second
)

// ActionApprovalRequest represents the request body for POST /api/client/v1/action-approval
type ActionApprovalRequest struct {
	ID        string               `json:"id"`
	UserID    string               `json:"user_id"`
	Request   ActionRequestDetails `json:"request"`
	CreatedAt time.Time            `json:"created_at"`
	Status    string               `json:"status"`
}

// ActionRequestDetails contains details about the risky action
type ActionRequestDetails struct {
	Method     string                 `json:"method"`
	ClientName string                 `json:"client_name"`
	Risk       ActionRiskInfo         `json:"risk"`
	Params     map[string]interface{} `json:"params"`
}

// ActionRiskInfo contains risk detection details for the approval request
type ActionRiskInfo struct {
	IsRisky     bool   `json:"is_risky"`
	Reason      string `json:"reason,omitempty"`
	MatchedRule string `json:"matched_rule,omitempty"`
}

// ActionApprovalResponse represents the response from GET /api/client/v1/action-approval/{id}
type ActionApprovalResponse struct {
	ID        string                  `json:"id"`
	Status    string                  `json:"status"` // "pending", "approved", "denied"
	Response  *ActionApprovalDecision `json:"response,omitempty"`
	CreatedAt UnixTime                `json:"created_at"`
}

// UnixTime handles Unix timestamp (float) or RFC3339 string
type UnixTime struct {
	time.Time
}

// UnmarshalJSON handles both Unix timestamp (float) and RFC3339 string
func (ut *UnixTime) UnmarshalJSON(data []byte) error {
	// Try parsing as float (Unix timestamp)
	var timestamp float64
	if err := json.Unmarshal(data, &timestamp); err == nil {
		ut.Time = time.Unix(int64(timestamp), int64((timestamp-float64(int64(timestamp)))*1e9))
		return nil
	}

	// Try parsing as RFC3339 string
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return err
	}

	ut.Time = t
	return nil
}

// ActionApprovalDecision represents the approval decision
type ActionApprovalDecision struct {
	Approved  bool     `json:"approved"`
	Responder string   `json:"responder"`
	Timestamp UnixTime `json:"timestamp"`
	Comment   *string  `json:"comment,omitempty"`
}

// AponoActionApprover submits approval requests to the Apono action-approval API and polls for results
type AponoActionApprover struct {
	baseURL    string
	httpClient *http.Client
	userID     string
	timeout    time.Duration
}

// NewAponoActionApprover creates a new action-approval API based approver
func NewAponoActionApprover(baseURL string, httpClient *http.Client, userID string, timeout time.Duration) *AponoActionApprover {
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	return &AponoActionApprover{
		baseURL:    baseURL,
		httpClient: httpClient,
		userID:     userID,
		timeout:    timeout,
	}
}

// RequestApproval creates an action-approval request and polls for the response
func (a *AponoActionApprover) RequestApproval(ctx context.Context, req ApprovalRequest) (bool, error) {
	approvalID := uuid.New().String()

	apiReq := ActionApprovalRequest{
		ID:     approvalID,
		UserID: a.userID,
		Request: ActionRequestDetails{
			Method:     "tools/call",
			ClientName: a.userID,
			Risk: ActionRiskInfo{
				IsRisky:     true,
				Reason:      req.Reason,
				MatchedRule: req.MatchedRule,
			},
			Params: map[string]interface{}{
				"name":      req.ToolName,
				"arguments": req.Arguments,
			},
		},
		CreatedAt: time.Now(),
		Status:    StatusPending,
	}

	reqJSON, _ := json.Marshal(apiReq)
	utils.McpLogf("[Approver] POST action-approval: user_id=%s, url=%s/api/client/v1/action-approval", a.userID, a.baseURL)
	utils.McpLogf("[Approver] Request body: %s", string(reqJSON))

	if err := a.createApproval(ctx, apiReq); err != nil {
		return false, fmt.Errorf("failed to create approval request: %w", err)
	}

	utils.McpLogf("[Approver] Action-approval request created: id=%s, waiting for response (timeout: %v)...", approvalID, a.timeout)

	return a.pollForApproval(ctx, approvalID)
}

// createApproval sends POST /api/client/v1/action-approval
func (a *AponoActionApprover) createApproval(ctx context.Context, req ActionApprovalRequest) error {
	url := fmt.Sprintf("%s/api/client/v1/action-approval", a.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	utils.McpLogf("[Approver] POST response: status=%d body=%s", resp.StatusCode, string(respBody))

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// getApprovalStatus sends GET /api/client/v1/action-approval/{id}
func (a *AponoActionApprover) getApprovalStatus(ctx context.Context, approvalID string) (*ActionApprovalResponse, error) {
	url := fmt.Sprintf("%s/api/client/v1/action-approval/%s", a.baseURL, approvalID)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var response ActionApprovalResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// pollForApproval polls the action-approval status until approved, denied, or timeout
func (a *AponoActionApprover) pollForApproval(ctx context.Context, approvalID string) (bool, error) {
	ticker := time.NewTicker(defaultPollInterval)
	defer ticker.Stop()

	timeout := time.After(a.timeout)

	for {
		select {
		case <-ctx.Done():
			return false, fmt.Errorf("approval request cancelled: %w", ctx.Err())

		case <-timeout:
			utils.McpLogf("[Approver] Approval request timed out: id=%s", approvalID)
			return false, fmt.Errorf("approval request timed out after %v", a.timeout)

		case <-ticker.C:
			response, err := a.getApprovalStatus(ctx, approvalID)
			if err != nil {
				utils.McpLogf("[Approver] Error checking approval status: %v", err)
				continue // Keep polling
			}

			switch response.Status {
			case StatusApproved:
				responder := ""
				if response.Response != nil {
					responder = response.Response.Responder
				}
				utils.McpLogf("[Approver] Approved by %s!", responder)
				return true, nil

			case StatusDenied:
				responder := ""
				if response.Response != nil {
					responder = response.Response.Responder
				}
				utils.McpLogf("[Approver] Denied by %s!", responder)
				return false, nil

			case StatusPending:
				utils.McpLogf("[Approver] Still pending, continuing to poll...")
				continue

			default:
				utils.McpLogf("[Approver] Unknown status %q, continuing to poll...", response.Status)
				continue
			}
		}
	}
}
