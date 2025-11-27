package notifier

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/apono-io/apono-cli/pkg/commands/mcp-proxy/approval"
	"github.com/apono-io/apono-cli/pkg/commands/mcp-proxy/auditor"
	"github.com/slack-go/slack"
)

// SlackNotifier sends approval requests to Slack
type SlackNotifier struct {
	client    *slack.Client
	channelID string
	userID    string // If set, will open DM with this user instead of posting to channel
}

// NewSlackNotifier creates a new Slack notifier
// If userID is provided, messages will be sent as DMs to that user
// Otherwise, messages will be sent to the specified channelID
func NewSlackNotifier(botToken, channelID, userID string) *SlackNotifier {
	return &SlackNotifier{
		client:    slack.New(botToken),
		channelID: channelID,
		userID:    userID,
	}
}

// SendApprovalRequest sends an approval request to Slack and returns the message timestamp
func (sn *SlackNotifier) SendApprovalRequest(ctx context.Context, req approval.ApprovalRequest) (string, error) {
	blocks := sn.buildApprovalBlocks(req)

	// Determine target: user DM or channel
	target := sn.channelID
	if sn.userID != "" {
		// Open a DM conversation with the user
		conversation, _, _, err := sn.client.OpenConversationContext(ctx, &slack.OpenConversationParameters{
			Users: []string{sn.userID},
		})
		if err != nil {
			return "", fmt.Errorf("failed to open DM conversation with user %s: %w", sn.userID, err)
		}
		target = conversation.ID
	}

	// Send message to Slack
	channelID, timestamp, err := sn.client.PostMessageContext(
		ctx,
		target,
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionText("Risky MCP Operation Detected", false),
	)

	if err != nil {
		return "", fmt.Errorf("failed to post message to Slack: %w", err)
	}

	// Return message timestamp as correlation ID
	return fmt.Sprintf("%s:%s", channelID, timestamp), nil
}

// buildApprovalBlocks creates Slack Block Kit blocks for the approval request
func (sn *SlackNotifier) buildApprovalBlocks(req approval.ApprovalRequest) []slack.Block {
	blocks := []slack.Block{}

	// Header
	headerText := slack.NewTextBlockObject(slack.PlainTextType, "🚨 Risky MCP Operation Detected", false, false)
	headerBlock := slack.NewHeaderBlock(headerText)
	blocks = append(blocks, headerBlock)

	// Risk information section
	riskFields := []*slack.TextBlockObject{
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Method:*\n%s", req.Method), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Risk Level:*\n%s", formatRiskLevel(req.Risk.Level)), false, false),
	}

	if req.ClientName != "" {
		riskFields = append(riskFields, slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Client:*\n%s", req.ClientName), false, false))
	}

	riskFields = append(riskFields, slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Reason:*\n%s", req.Risk.Reason), false, false))

	riskSection := slack.NewSectionBlock(nil, riskFields, nil)
	blocks = append(blocks, riskSection)

	// Matched rule
	if req.Risk.MatchedRule != "" {
		ruleText := slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Matched Rule:* `%s`", req.Risk.MatchedRule), false, false)
		ruleSection := slack.NewSectionBlock(ruleText, nil, nil)
		blocks = append(blocks, ruleSection)
	}

	// Parameters section (if present)
	if len(req.Params) > 0 {
		paramsJSON, err := json.MarshalIndent(req.Params, "", "  ")
		if err == nil {
			paramsText := slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Parameters:*\n```%s```", string(paramsJSON)), false, false)
			paramsSection := slack.NewSectionBlock(paramsText, nil, nil)
			blocks = append(blocks, paramsSection)
		}
	}

	// Timestamp
	timestampText := slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Time:* %s", req.Timestamp.Format(time.RFC3339)), false, false)
	timestampSection := slack.NewSectionBlock(timestampText, nil, nil)
	blocks = append(blocks, timestampSection)

	// Divider
	blocks = append(blocks, slack.NewDividerBlock())

	// Action buttons
	approveButton := slack.NewButtonBlockElement(
		"approve",
		req.ID,
		slack.NewTextBlockObject(slack.PlainTextType, "✅ Approve", false, false),
	)
	approveButton.Style = slack.StylePrimary

	denyButton := slack.NewButtonBlockElement(
		"deny",
		req.ID,
		slack.NewTextBlockObject(slack.PlainTextType, "❌ Deny", false, false),
	)
	denyButton.Style = slack.StyleDanger

	actionsBlock := slack.NewActionBlock(req.ID, approveButton, denyButton)
	blocks = append(blocks, actionsBlock)

	return blocks
}

// formatRiskLevel converts risk level to a readable string
func formatRiskLevel(level interface{}) string {
	// Handle both int and custom RiskLevel types
	var levelInt int
	switch v := level.(type) {
	case auditor.RiskLevel:
		levelInt = int(v)
	case int:
		levelInt = v
	default:
		return "UNKNOWN"
	}

	switch levelInt {
	case 0:
		return "NONE"
	case 1:
		return "LOW"
	case 2:
		return "MEDIUM"
	case 3:
		return "HIGH"
	default:
		return "UNKNOWN"
	}
}
