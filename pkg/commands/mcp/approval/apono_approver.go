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
	Mode      string   `json:"mode,omitempty"`    // "approve_once", "approve_intent", "approve_pattern"
	Pattern   string   `json:"pattern,omitempty"` // pattern when mode is approve_pattern
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
func (a *AponoActionApprover) RequestApproval(ctx context.Context, req ApprovalRequest) (*ApprovalResult, error) {
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
				"name":              req.ToolName,
				"arguments":         truncateArguments(req.Arguments),
				"intent":            req.Intent,
				"suggested_pattern": req.SuggestedPattern,
			},
		},
		CreatedAt: time.Now(),
		Status:    StatusPending,
	}

	reqJSON, _ := json.Marshal(apiReq)
	utils.McpLogf("[Approver] POST action-approval: user_id=%s, url=%s/api/client/v1/action-approval", a.userID, a.baseURL)
	utils.McpLogf("[Approver] Request body: %s", string(reqJSON))

	createResp, err := a.createApproval(ctx, apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create approval request: %w", err)
	}

	// Check if the POST response already contains a terminal status (e.g., auto-approved via intent matching)
	if createResp != nil {
		switch createResp.Status {
		case StatusApproved:
			utils.McpLogf("[Approver] Auto-approved on create: id=%s", approvalID)
			return a.buildResult(createResp), nil
		case StatusDenied:
			utils.McpLogf("[Approver] Auto-denied on create: id=%s", approvalID)
			return &ApprovalResult{Approved: false, Mode: ApprovalModeDeny}, nil
		}
	}

	utils.McpLogf("[Approver] Action-approval request created: id=%s, waiting for response (timeout: %v)...", approvalID, a.timeout)

	return a.pollForApproval(ctx, approvalID)
}

// createApproval sends POST /api/client/v1/action-approval and returns the parsed response
func (a *AponoActionApprover) createApproval(ctx context.Context, req ActionApprovalRequest) (*ActionApprovalResponse, error) {
	url := fmt.Sprintf("%s/api/client/v1/action-approval", a.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	utils.McpLogf("[Approver] POST response: status=%d body=%s", resp.StatusCode, string(respBody))

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response to detect auto-approval (e.g., intent match)
	var response ActionApprovalResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		utils.McpLogf("[Approver] Could not parse POST response body, will poll: %v", err)
		return nil, nil
	}

	return &response, nil
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
func (a *AponoActionApprover) pollForApproval(ctx context.Context, approvalID string) (*ApprovalResult, error) {
	ticker := time.NewTicker(defaultPollInterval)
	defer ticker.Stop()

	timeout := time.After(a.timeout)

	pollCount := 0
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("approval request cancelled: %w", ctx.Err())

		case <-timeout:
			utils.McpLogf("[Approver] Approval request timed out: id=%s (polled %d times over %v)", approvalID, pollCount, a.timeout)
			return nil, fmt.Errorf("approval request timed out after %v (%d polls)", a.timeout, pollCount)

		case <-ticker.C:
			pollCount++
			response, err := a.getApprovalStatus(ctx, approvalID)
			if err != nil {
				utils.McpLogf("[Approver] Error checking approval status (poll #%d): %v", pollCount, err)
				continue // Keep polling
			}

			switch response.Status {
			case StatusApproved:
				responder := ""
				mode := ""
				if response.Response != nil {
					responder = response.Response.Responder
					mode = response.Response.Mode
				}
				result := a.buildResult(response)
				utils.McpLogf("[Approver] Approved by %s (mode=%s, pattern=%s) after %d polls", responder, mode, result.Pattern, pollCount)
				return result, nil

			case StatusDenied:
				responder := ""
				if response.Response != nil {
					responder = response.Response.Responder
				}
				utils.McpLogf("[Approver] Denied by %s after %d polls", responder, pollCount)
				return &ApprovalResult{Approved: false, Mode: ApprovalModeDeny}, nil

			case StatusPending:
				if pollCount%10 == 1 {
					utils.McpLogf("[Approver] Still pending id=%s (poll #%d, ~%ds elapsed)...", approvalID, pollCount, pollCount*3)
				}
				continue

			default:
				utils.McpLogf("[Approver] Unknown status %q (poll #%d), continuing...", response.Status, pollCount)
				continue
			}
		}
	}
}

// buildResult converts an API response to an ApprovalResult, handling backward compat
func (a *AponoActionApprover) buildResult(resp *ActionApprovalResponse) *ApprovalResult {
	result := &ApprovalResult{Approved: true, Mode: ApprovalModeApproveOnce}

	if resp.Response != nil {
		switch ApprovalMode(resp.Response.Mode) {
		case ApprovalModeApproveIntent:
			result.Mode = ApprovalModeApproveIntent
		case ApprovalModeApprovePattern:
			result.Mode = ApprovalModeApprovePattern
			result.Pattern = resp.Response.Pattern
		}
	}

	return result
}

const maxArgumentValueLength = 500

// truncateArguments returns a copy of the arguments map with long string values truncated.
// This prevents Slack API errors when the backend renders approval messages with action payloads
// that exceed Slack's block kit size limits.
func truncateArguments(args map[string]interface{}) map[string]interface{} {
	if args == nil {
		return nil
	}

	truncated := make(map[string]interface{}, len(args))
	for k, v := range args {
		switch val := v.(type) {
		case string:
			if len(val) > maxArgumentValueLength {
				truncated[k] = val[:maxArgumentValueLength] + fmt.Sprintf("... (truncated, %d chars total)", len(val))
			} else {
				truncated[k] = val
			}
		default:
			truncated[k] = v
		}
	}

	return truncated
}
