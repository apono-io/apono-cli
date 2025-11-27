package notifier

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/apono-io/apono-cli/pkg/commands/mcp-proxy/approval"
	"github.com/apono-io/apono-cli/pkg/utils"
	"github.com/slack-go/slack"
)

// CallbackHandler handles Slack interactive messages
type CallbackHandler struct {
	signingSecret    string
	skipVerification bool
	approvalStore    *approval.InMemoryApprovalStore
}

// NewCallbackHandler creates a new callback handler
func NewCallbackHandler(signingSecret string, skipVerification bool, approvalStore *approval.InMemoryApprovalStore) *CallbackHandler {
	return &CallbackHandler{
		signingSecret:    signingSecret,
		skipVerification: skipVerification,
		approvalStore:    approvalStore,
	}
}

// HandleInteraction processes Slack interactive messages
func (h *CallbackHandler) HandleInteraction(w http.ResponseWriter, r *http.Request) {
	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		utils.McpLogf("Error reading request body: %v", err)
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Verify Slack request signature (unless skipped)
	if !h.skipVerification {
		if !h.verifySlackRequest(r.Header, body) {
			utils.McpLogf("Invalid Slack signature")
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
	} else {
		utils.McpLogf("WARNING: Slack signature verification is DISABLED")
	}

	// Parse the form data from the body
	values, err := url.ParseQuery(string(body))
	if err != nil {
		utils.McpLogf("Error parsing form data: %v", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Parse the payload
	var payload slack.InteractionCallback
	if err := json.Unmarshal([]byte(values.Get("payload")), &payload); err != nil {
		utils.McpLogf("Error parsing payload: %v", err)
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Process the action
	if len(payload.ActionCallback.BlockActions) == 0 {
		utils.McpLogf("No actions found in payload")
		http.Error(w, "No actions found", http.StatusBadRequest)
		return
	}

	action := payload.ActionCallback.BlockActions[0]
	approvalID := action.Value
	actionID := action.ActionID

	utils.McpLogf("Received Slack action: %s for approval ID: %s", actionID, approvalID)

	// Determine approval decision
	var approved bool
	switch actionID {
	case "approve":
		approved = true
	case "deny":
		approved = false
	default:
		utils.McpLogf("Unknown action ID: %s", actionID)
		http.Error(w, "Unknown action", http.StatusBadRequest)
		return
	}

	// Create approval response
	response := approval.ApprovalResponse{
		Approved:  approved,
		Responder: payload.User.Name,
		Timestamp: time.Now(),
		Comment:   "",
	}

	// Update the approval store
	if err := h.approvalStore.UpdateResponse(approvalID, response); err != nil {
		utils.McpLogf("Error updating approval store: %v", err)
		// Don't return error to Slack - the update might have already happened (first responder wins)
	}

	// Respond to Slack with updated message
	responseMsg := h.buildResponseMessage(approved, payload.User.Name)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseMsg)

	utils.McpLogf("Approval %s by %s for ID: %s", map[bool]string{true: "granted", false: "denied"}[approved], payload.User.Name, approvalID)
}

// verifySlackRequest verifies the Slack request signature
func (h *CallbackHandler) verifySlackRequest(headers http.Header, body []byte) bool {
	timestamp := headers.Get("X-Slack-Request-Timestamp")
	signature := headers.Get("X-Slack-Signature")

	utils.McpLogf("=== Slack Signature Verification Debug ===")
	utils.McpLogf("Timestamp header: %s", timestamp)
	utils.McpLogf("Signature header: %s", signature)
	utils.McpLogf("Body length: %d bytes", len(body))
	utils.McpLogf("Body content: %s", string(body))
	utils.McpLogf("Signing secret: %s", h.signingSecret)

	if timestamp == "" || signature == "" {
		utils.McpLogf("ERROR: Missing timestamp or signature header")
		return false
	}

	// Check if timestamp is within 5 minutes
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		utils.McpLogf("ERROR: Failed to parse timestamp: %v", err)
		return false
	}

	currentTime := time.Now().Unix()
	timeDiff := currentTime - ts
	utils.McpLogf("Current time: %d, Request time: %d, Diff: %d seconds", currentTime, ts, timeDiff)

	if timeDiff > 300 {
		utils.McpLogf("ERROR: Request too old (>5 minutes)")
		return false
	}

	// Calculate expected signature
	sigBaseString := fmt.Sprintf("v0:%s:%s", timestamp, string(body))
	utils.McpLogf("Signature base string: %s", sigBaseString)

	mac := hmac.New(sha256.New, []byte(h.signingSecret))
	mac.Write([]byte(sigBaseString))
	expectedSignature := "v0=" + hex.EncodeToString(mac.Sum(nil))

	utils.McpLogf("Expected signature: %s", expectedSignature)
	utils.McpLogf("Received signature: %s", signature)

	match := hmac.Equal([]byte(expectedSignature), []byte(signature))
	utils.McpLogf("Signatures match: %v", match)
	utils.McpLogf("=== End Debug ===")

	return match
}

// buildResponseMessage creates a Slack message response for the interaction
func (h *CallbackHandler) buildResponseMessage(approved bool, responder string) map[string]interface{} {
	var statusEmoji, statusText string

	if approved {
		statusEmoji = "✅"
		statusText = "APPROVED"
	} else {
		statusEmoji = "❌"
		statusText = "DENIED"
	}

	return map[string]interface{}{
		"replace_original": true,
		"blocks": []map[string]interface{}{
			{
				"type": "header",
				"text": map[string]interface{}{
					"type": "plain_text",
					"text": fmt.Sprintf("%s Request %s", statusEmoji, statusText),
				},
			},
			{
				"type": "section",
				"text": map[string]interface{}{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*Decision by:* %s\n*Time:* %s", responder, time.Now().Format(time.RFC3339)),
				},
			},
			{
				"type": "context",
				"elements": []map[string]interface{}{
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("Status: *%s*", statusText),
					},
				},
			},
		},
	}
}
