package slack

import (
	"context"
	"fmt"
	"net/http"

	"github.com/slack-go/slack"
)

// Client wraps the Slack API client
type Client struct {
	api           *slack.Client
	signingSecret string
}

// NewClient creates a new Slack client
func NewClient(botToken, signingSecret string) *Client {
	return &Client{
		api:           slack.New(botToken),
		signingSecret: signingSecret,
	}
}

// GetUserInfo retrieves user information from Slack
func (c *Client) GetUserInfo(ctx context.Context, userID string) (*slack.User, error) {
	user, err := c.api.GetUserInfoContext(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	return user, nil
}

// GetUserEmail retrieves a user's email from Slack
func (c *Client) GetUserEmail(ctx context.Context, userID string) (string, error) {
	user, err := c.GetUserInfo(ctx, userID)
	if err != nil {
		return "", err
	}
	return user.Profile.Email, nil
}

// GetUsername retrieves a user's name from Slack
func (c *Client) GetUsername(ctx context.Context, userID string) (string, error) {
	user, err := c.GetUserInfo(ctx, userID)
	if err != nil {
		return "", err
	}

	// Prefer real name, fall back to display name, then username
	if user.RealName != "" {
		return user.RealName, nil
	}
	if user.Profile.DisplayName != "" {
		return user.Profile.DisplayName, nil
	}
	return user.Name, nil
}

// PostMessage sends a message to a Slack channel or user
// Returns channel and timestamp separately for use with UpdateMessage
func (c *Client) PostMessage(ctx context.Context, channelID string, options ...slack.MsgOption) (channel string, timestamp string, err error) {
	channel, timestamp, err = c.api.PostMessageContext(ctx, channelID, options...)
	if err != nil {
		return "", "", fmt.Errorf("failed to post message: %w", err)
	}
	return channel, timestamp, nil
}

// PostEphemeral sends an ephemeral message (visible only to one user)
func (c *Client) PostEphemeral(ctx context.Context, channelID, userID string, options ...slack.MsgOption) error {
	_, err := c.api.PostEphemeralContext(ctx, channelID, userID, options...)
	if err != nil {
		return fmt.Errorf("failed to post ephemeral message: %w", err)
	}
	return nil
}

// UpdateMessage updates an existing message
func (c *Client) UpdateMessage(ctx context.Context, channelID, timestamp string, options ...slack.MsgOption) error {
	_, _, _, err := c.api.UpdateMessageContext(ctx, channelID, timestamp, options...)
	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}
	return nil
}

// VerifySignature verifies a Slack request signature
func (c *Client) VerifySignature(headers http.Header, body string) error {
	sv, err := slack.NewSecretsVerifier(headers, c.signingSecret)
	if err != nil {
		return fmt.Errorf("failed to create secrets verifier: %w", err)
	}

	if _, err := sv.Write([]byte(body)); err != nil {
		return fmt.Errorf("failed to write body to verifier: %w", err)
	}

	if err := sv.Ensure(); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// OpenView opens a modal view in Slack
func (c *Client) OpenView(triggerID string, view slack.ModalViewRequest) (*slack.ViewResponse, error) {
	resp, err := c.api.OpenView(triggerID, view)
	if err != nil {
		return nil, fmt.Errorf("failed to open view: %w", err)
	}
	return resp, nil
}

// UpdateView updates an existing modal view (for validation errors)
func (c *Client) UpdateView(viewID, hash string, view slack.ModalViewRequest) (*slack.ViewResponse, error) {
	resp, err := c.api.UpdateView(view, "", hash, viewID)
	if err != nil {
		return nil, fmt.Errorf("failed to update view: %w", err)
	}
	return resp, nil
}

// GetUserGroupMembers retrieves the list of user IDs in a Slack user group
func (c *Client) GetUserGroupMembers(ctx context.Context, groupID string) ([]string, error) {
	members, err := c.api.GetUserGroupMembersContext(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user group members: %w", err)
	}
	return members, nil
}

// GetUserGroupHandle retrieves the handle (e.g., @security-team) for a Slack user group
func (c *Client) GetUserGroupHandle(ctx context.Context, groupID string) (string, error) {
	userGroups, err := c.api.GetUserGroupsContext(ctx, slack.GetUserGroupsOptionIncludeUsers(false))
	if err != nil {
		return "", fmt.Errorf("failed to get user groups: %w", err)
	}

	for _, group := range userGroups {
		if group.ID == groupID {
			return group.Handle, nil
		}
	}

	return "", fmt.Errorf("user group not found: %s", groupID)
}
