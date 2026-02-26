package slack

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/slack-go/slack"
)

// InteractionHandler handles Slack interactive message callbacks
type InteractionHandler struct {
	client *Client
}

// NewInteractionHandler creates a new interaction handler
func NewInteractionHandler(client *Client) *InteractionHandler {
	return &InteractionHandler{client: client}
}

// InteractionCallback represents a Slack interaction callback
type InteractionCallback struct {
	Type        string                       `json:"type"`
	User        slack.User                   `json:"user"`
	ResponseURL string                       `json:"response_url"`
	TriggerID   string                       `json:"trigger_id"`
	Actions     []slack.BlockAction          `json:"actions"`
	Container   slack.Container              `json:"container"`
	Channel     slack.Channel                `json:"channel"`
	Message     slack.Message                `json:"message"`
	View        slack.View                   `json:"view"`
	Team        slack.Team                   `json:"team"`
	Token       string                       `json:"token"`
}

// ParseInteractionCallback parses a Slack interaction callback from JSON
func ParseInteractionCallback(payload string) (*InteractionCallback, error) {
	var callback InteractionCallback
	err := json.Unmarshal([]byte(payload), &callback)
	if err != nil {
		return nil, fmt.Errorf("failed to parse interaction callback: %w", err)
	}
	return &callback, nil
}

// GetActionValue gets the value of the first action in the callback
func (c *InteractionCallback) GetActionValue() string {
	if len(c.Actions) > 0 {
		return c.Actions[0].Value
	}
	return ""
}

// GetActionID gets the ID of the first action in the callback
func (c *InteractionCallback) GetActionID() string {
	if len(c.Actions) > 0 {
		return c.Actions[0].ActionID
	}
	return ""
}

// IsApproval checks if the interaction is an approval action
func (c *InteractionCallback) IsApproval() bool {
	return c.GetActionID() == "approve"
}

// IsDenial checks if the interaction is a denial action
func (c *InteractionCallback) IsDenial() bool {
	return c.GetActionID() == "deny"
}

// UpdateMessageWithResult updates the original message with the approval/denial result
func (h *InteractionHandler) UpdateMessageWithResult(ctx context.Context, callback *InteractionCallback, approved bool, approver string) error {
	var statusText string
	var statusEmoji string

	if approved {
		statusText = "Approved"
		statusEmoji = "✅"
	} else {
		statusText = "Denied"
		statusEmoji = "❌"
	}

	// Get the original message blocks
	originalBlocks := callback.Message.Blocks.BlockSet

	// Find and update the section block
	var updatedBlocks []slack.Block
	for _, block := range originalBlocks {
		if section, ok := block.(*slack.SectionBlock); ok {
			// Add approval status to the section
			section.Text.Text = section.Text.Text + fmt.Sprintf("\n\n%s *%s* by <@%s>",
				statusEmoji, statusText, approver)
			updatedBlocks = append(updatedBlocks, section)
		}
	}

	// Remove action buttons (they're no longer needed)
	// Don't include the action block in updatedBlocks

	err := h.client.UpdateMessage(
		ctx,
		callback.Channel.ID,
		callback.Message.Timestamp,
		slack.MsgOptionBlocks(updatedBlocks...),
	)
	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	return nil
}

// RespondToInteraction sends a response to a Slack interaction
func (h *InteractionHandler) RespondToInteraction(ctx context.Context, responseURL, message string) error {
	// This would typically use the response_url to send a response
	// For now, we'll use a simple approach
	// In production, you'd want to use http.Post to the response_url
	return nil
}
