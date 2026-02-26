package slack

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"github.com/ssm-access-manager/internal/models"
)

// Notifier handles sending notifications to Slack
type Notifier struct {
	client *Client
}

// NewNotifier creates a new Slack notifier
func NewNotifier(client *Client) *Notifier {
	return &Notifier{client: client}
}

// SendApprovalRequest sends an approval request to a manager
func (n *Notifier) SendApprovalRequest(ctx context.Context, managerID string, request *models.AccessRequest) error {
	blocks := []slack.Block{
		slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: fmt.Sprintf("*New Access Request*\n\n"+
					"*User:* <@%s>\n"+
					"*Host:* `%s`\n"+
					"*Port:* `%d`\n"+
					"*Account:* `%s`\n"+
					"*Expires:* %s\n"+
					"*Request ID:* `%s`",
					request.UserID,
					request.Host,
					request.Port,
					request.AccountID,
					request.ExpirationDate.Format("2006-01-02 15:04 MST"),
					request.RequestID,
				),
			},
			nil,
			nil,
		),
		slack.NewActionBlock(
			"approval_actions",
			slack.NewButtonBlockElement(
				"approve",
				request.RequestID,
				&slack.TextBlockObject{Type: slack.PlainTextType, Text: "Approve"},
			).WithStyle(slack.StylePrimary),
			slack.NewButtonBlockElement(
				"deny",
				request.RequestID,
				&slack.TextBlockObject{Type: slack.PlainTextType, Text: "Deny"},
			).WithStyle(slack.StyleDanger),
		),
	}

	_, _, err := n.client.PostMessage(ctx, managerID, slack.MsgOptionBlocks(blocks...))
	if err != nil {
		return fmt.Errorf("failed to send approval request: %w", err)
	}

	return nil
}

// SendApprovalConfirmation sends a confirmation to the user that their request was approved
func (n *Notifier) SendApprovalConfirmation(ctx context.Context, userID string, request *models.AccessRequest) error {
	var message string

	// Check if this is a two-tier approval request
	if request.ManagerGroupID != "" && request.SecurityApproverName != nil && request.ManagerApproverName != nil {
		// Two-tier approval - show both approvers
		message = fmt.Sprintf("✅ *Access Request Fully Approved*\n\n"+
			"Your request for access to `%s:%d` has been fully approved.\n"+
			"The SSM document will be created shortly.\n\n"+
			"*Reason:* %s\n"+
			"*Security Approval:* %s\n"+
			"*Manager Approval:* %s\n"+
			"*Expires:* %s\n"+
			"*Request ID:* `%s`",
			request.Host,
			request.Port,
			request.Reason,
			*request.SecurityApproverName,
			*request.ManagerApproverName,
			request.ExpirationDate.Format("2006-01-02 15:04 MST"),
			request.RequestID,
		)
	} else if request.Approver != nil {
		// Legacy single approval
		message = fmt.Sprintf("✅ *Access Request Approved*\n\n"+
			"Your request for access to `%s:%d` has been approved by %s.\n"+
			"The SSM document will be created shortly.\n\n"+
			"*Reason:* %s\n"+
			"*Expires:* %s\n"+
			"*Request ID:* `%s`",
			request.Host,
			request.Port,
			*request.Approver,
			request.Reason,
			request.ExpirationDate.Format("2006-01-02 15:04 MST"),
			request.RequestID,
		)
	} else {
		// Fallback
		message = fmt.Sprintf("✅ *Access Request Approved*\n\n"+
			"Your request for access to `%s:%d` has been approved.\n"+
			"The SSM document will be created shortly.\n\n"+
			"*Reason:* %s\n"+
			"*Expires:* %s\n"+
			"*Request ID:* `%s`",
			request.Host,
			request.Port,
			request.Reason,
			request.ExpirationDate.Format("2006-01-02 15:04 MST"),
			request.RequestID,
		)
	}

	_, _, err := n.client.PostMessage(ctx, userID, slack.MsgOptionText(message, false))
	if err != nil {
		return fmt.Errorf("failed to send approval confirmation: %w", err)
	}

	return nil
}

// SendDenialNotification sends a notification to the user that their request was denied
func (n *Notifier) SendDenialNotification(ctx context.Context, userID string, request *models.AccessRequest, reason string) error {
	message := fmt.Sprintf("❌ *Access Request Denied*\n\n"+
		"Your request for access to `%s:%d` has been denied by %s.\n\n"+
		"*Reason:* %s\n"+
		"*Request ID:* `%s`",
		request.Host,
		request.Port,
		*request.Approver,
		reason,
		request.RequestID,
	)

	_, _, err := n.client.PostMessage(ctx, userID, slack.MsgOptionText(message, false))
	if err != nil {
		return fmt.Errorf("failed to send denial notification: %w", err)
	}

	return nil
}

// SendExpirationNotification sends a notification to the user that their access has expired
func (n *Notifier) SendExpirationNotification(ctx context.Context, userID string, document *models.SSMDocument) error {
	message := fmt.Sprintf("⏰ *Access Expired*\n\n"+
		"Your access to `%s:%d` has expired and the SSM document has been removed.\n\n"+
		"*Document:* `%s`\n"+
		"*Expired:* %s",
		document.Host,
		document.Port,
		document.DocumentName,
		document.ExpiresAt.Format("2006-01-02 15:04 MST"),
	)

	_, _, err := n.client.PostMessage(ctx, userID, slack.MsgOptionText(message, false))
	if err != nil {
		return fmt.Errorf("failed to send expiration notification: %w", err)
	}

	return nil
}

// SendErrorNotification sends an error notification to the user
func (n *Notifier) SendErrorNotification(ctx context.Context, userID, errorMessage string) error {
	message := fmt.Sprintf("❌ *Error*\n\n%s", errorMessage)

	_, _, err := n.client.PostMessage(ctx, userID, slack.MsgOptionText(message, false))
	if err != nil {
		return fmt.Errorf("failed to send error notification: %w", err)
	}

	return nil
}

// SendRequestConfirmation sends a confirmation to the user that their request was submitted
func (n *Notifier) SendRequestConfirmation(ctx context.Context, userID string, request *models.AccessRequest) error {
	// Build message based on whether it's a new request with manager group or legacy
	var message string
	
	if request.ManagerGroupID != "" && request.ManagerGroupName != "" {
		// New request with manager group selection
		message = fmt.Sprintf("✅ *Access Request Submitted*\n\n"+
			"Your request for access to `%s:%d` has been submitted for approval.\n\n"+
			"*Account:* `%s`\n"+
			"*Expires:* %s\n"+
			"*Request ID:* `%s`\n\n"+
			"*Required Approvals:*\n"+
			"• Security approval\n"+
			"• Manager approval from %s\n\n"+
			"You will be notified as approvals are granted.",
			request.Host,
			request.Port,
			request.AccountID,
			request.ExpirationDate.Format("2006-01-02 15:04 MST"),
			request.RequestID,
			request.ManagerGroupName,
		)
	} else {
		// Legacy request
		message = fmt.Sprintf("✅ *Access Request Submitted*\n\n"+
			"Your request for access to `%s:%d` has been submitted for approval.\n\n"+
			"*Account:* `%s`\n"+
			"*Expires:* %s\n"+
			"*Request ID:* `%s`\n\n"+
			"You will be notified once a manager reviews your request.",
			request.Host,
			request.Port,
			request.AccountID,
			request.ExpirationDate.Format("2006-01-02 15:04 MST"),
			request.RequestID,
		)
	}

	_, _, err := n.client.PostMessage(ctx, userID, slack.MsgOptionText(message, false))
	if err != nil {
		return fmt.Errorf("failed to send request confirmation: %w", err)
	}

	return nil
}

// SendMissingFieldPrompt sends a prompt to the user for missing required fields
func (n *Notifier) SendMissingFieldPrompt(ctx context.Context, userID string, missingFields []string) error {
	fieldsText := ""
	for _, field := range missingFields {
		fieldsText += fmt.Sprintf("• %s\n", field)
	}

	message := fmt.Sprintf("❌ *Missing Required Fields*\n\n"+
		"Your access request is missing the following required fields:\n\n"+
		"%s\n"+
		"Please submit your request again with all required fields.\n\n"+
		"*Usage:* `/ssm-access host=example.com port=8080 account=123456789012 expires=2024-12-31`",
		fieldsText,
	)

	_, _, err := n.client.PostMessage(ctx, userID, slack.MsgOptionText(message, false))
	if err != nil {
		return fmt.Errorf("failed to send missing field prompt: %w", err)
	}

	return nil
}

// SendDocumentCreationSuccess sends a notification that the SSM document was created
func (n *Notifier) SendDocumentCreationSuccess(ctx context.Context, userID string, document *models.SSMDocument, account *models.Account) error {
	// Build the connection command with bastion host ID if available
	var connectionCmd string
	if account != nil && account.HasBastionHost() {
		connectionCmd = fmt.Sprintf("aws ssm start-session --target %s --document-name %s --parameters portNumber=%d,host=%s",
			account.BastionHostID,
			document.DocumentName,
			document.Port,
			document.Host,
		)
	} else {
		// Fallback message if bastion host ID is not configured
		connectionCmd = "⚠️ Bastion host not configured for this account. Contact your administrator to configure the bastion host ID."
	}

	message := fmt.Sprintf("✅ *SSM Document Created*\n\n"+
		"Your SSM document has been created successfully.\n\n"+
		"*Document Name:* `%s`\n"+
		"*Host:* `%s`\n"+
		"*Port:* `%d`\n"+
		"*Account:* `%s`\n"+
		"*Region:* `%s`\n"+
		"*Expires:* %s\n\n"+
		"*Connection Command:*\n```\n%s\n```",
		document.DocumentName,
		document.Host,
		document.Port,
		document.AccountID,
		document.Region,
		document.ExpiresAt.Format("2006-01-02 15:04 MST"),
		connectionCmd,
	)

	_, _, err := n.client.PostMessage(ctx, userID, slack.MsgOptionText(message, false))
	if err != nil {
		return fmt.Errorf("failed to send document creation success: %w", err)
	}

	return nil
}

// SendDocumentCreationFailure sends a notification that the SSM document creation failed
func (n *Notifier) SendDocumentCreationFailure(ctx context.Context, userID string, request *models.AccessRequest, errorMsg string) error {
	message := fmt.Sprintf("❌ *SSM Document Creation Failed*\n\n"+
		"Failed to create SSM document for your approved request.\n\n"+
		"*Host:* `%s`\n"+
		"*Port:* `%d`\n"+
		"*Error:* %s\n"+
		"*Request ID:* `%s`\n\n"+
		"Please contact an administrator for assistance.",
		request.Host,
		request.Port,
		errorMsg,
		request.RequestID,
	)

	_, _, err := n.client.PostMessage(ctx, userID, slack.MsgOptionText(message, false))
	if err != nil {
		return fmt.Errorf("failed to send document creation failure: %w", err)
	}

	return nil
}

// SendAdminAlert sends an alert to administrators
func (n *Notifier) SendAdminAlert(ctx context.Context, adminUserIDs []string, alertMessage string) error {
	message := fmt.Sprintf("🚨 *Admin Alert*\n\n%s\n\n*Time:* %s",
		alertMessage,
		time.Now().Format("2006-01-02 15:04:05 MST"),
	)

	for _, adminID := range adminUserIDs {
		_, _, err := n.client.PostMessage(ctx, adminID, slack.MsgOptionText(message, false))
		if err != nil {
			// Log error but continue to notify other admins
			fmt.Printf("Failed to send admin alert to %s: %v\n", adminID, err)
		}
	}

	return nil
}

// SendRevocationNotification sends a notification to the user that their access was revoked
func (n *Notifier) SendRevocationNotification(ctx context.Context, userID string, request *models.AccessRequest, revokedBy, reason string) error {
	message := fmt.Sprintf("🚫 *Access Revoked*\n\n"+
		"Your access to `%s:%d` has been revoked by %s.\n\n"+
		"*Reason:* %s\n"+
		"*Request ID:* `%s`\n"+
		"*Revoked At:* %s\n\n"+
		"The SSM document has been deleted and you no longer have access.",
		request.Host,
		request.Port,
		revokedBy,
		reason,
		request.RequestID,
		time.Now().Format("2006-01-02 15:04 MST"),
	)

	_, _, err := n.client.PostMessage(ctx, userID, slack.MsgOptionText(message, false))
	if err != nil {
		return fmt.Errorf("failed to send revocation notification: %w", err)
	}

	return nil
}

// SendApprovalRequestToGroups sends approval request to all members of security and manager groups
// Returns a map of userID -> message timestamp for later updates
func (n *Notifier) SendApprovalRequestToGroups(ctx context.Context, request *models.AccessRequest, securityGroup *models.ApprovalGroup, managerGroup *models.ApprovalGroup, groupCache *GroupMembershipCache) (map[string]string, error) {
	// Get security group members
	securityMembers := []string{}
	if securityGroup != nil && groupCache != nil {
		members, err := groupCache.GetMembers(ctx, securityGroup.GroupID)
		if err != nil {
			fmt.Printf("Warning: Failed to get security group members: %v\n", err)
		} else {
			securityMembers = members
		}
	}
	
	// Get manager group members
	managerMembers := []string{}
	if managerGroup != nil && groupCache != nil {
		members, err := groupCache.GetMembers(ctx, managerGroup.GroupID)
		if err != nil {
			fmt.Printf("Warning: Failed to get manager group members: %v\n", err)
		} else {
			managerMembers = members
		}
	}
	
	// Deduplicate members (union of both groups)
	memberSet := make(map[string]bool)
	for _, member := range securityMembers {
		memberSet[member] = true
	}
	for _, member := range managerMembers {
		memberSet[member] = true
	}
	
	// Build approval message showing both required approvals
	securityGroupHandle := ""
	if securityGroup != nil {
		securityGroupHandle = securityGroup.SlackHandle
	}
	
	managerGroupHandle := ""
	if managerGroup != nil {
		managerGroupHandle = managerGroup.SlackHandle
	}
	
	blocks := []slack.Block{
		slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: fmt.Sprintf("*New Access Request - Two-Tier Approval Required*\n\n"+
					"*User:* <@%s>\n"+
					"*Host:* `%s`\n"+
					"*Port:* `%d`\n"+
					"*Account:* `%s`\n"+
					"*Expires:* %s\n"+
					"*Reason:* %s\n"+
					"*Request ID:* `%s`\n\n"+
					"*Required Approvals:*\n"+
					"• Security approval from %s\n"+
					"• Manager approval from %s",
					request.UserID,
					request.Host,
					request.Port,
					request.AccountID,
					request.ExpirationDate.Format("2006-01-02 15:04 MST"),
					request.Reason,
					request.RequestID,
					securityGroupHandle,
					managerGroupHandle,
				),
			},
			nil,
			nil,
		),
		slack.NewActionBlock(
			"approval_actions",
			slack.NewButtonBlockElement(
				"approve",
				request.RequestID,
				&slack.TextBlockObject{Type: slack.PlainTextType, Text: "Approve"},
			).WithStyle(slack.StylePrimary),
			slack.NewButtonBlockElement(
				"deny",
				request.RequestID,
				&slack.TextBlockObject{Type: slack.PlainTextType, Text: "Deny"},
			).WithStyle(slack.StyleDanger),
		),
	}
	
	// Send to all group members and collect timestamps
	timestamps := make(map[string]string)
	for memberID := range memberSet {
		channel, timestamp, err := n.client.PostMessage(ctx, memberID, slack.MsgOptionBlocks(blocks...))
		if err != nil {
			// Log error but continue to notify other members
			fmt.Printf("Warning: Failed to send approval request to %s: %v\n", memberID, err)
		} else {
			// Store the channel:timestamp for this user's message
			msgRef := fmt.Sprintf("%s:%s", channel, timestamp)
			timestamps[memberID] = msgRef
		}
	}
	
	return timestamps, nil
}

// SendApprovalStatusUpdate sends a status update to the requester about approval progress
func (n *Notifier) SendApprovalStatusUpdate(ctx context.Context, request *models.AccessRequest) error {
	var message string
	
	if request.IsApproved() {
		// Fully approved
		message = fmt.Sprintf("✅ *Access Request Fully Approved*\n\n"+
			"Your request for access to `%s:%d` has been fully approved.\n\n"+
			"*Security Approval:* %s at %s\n"+
			"*Manager Approval:* %s at %s\n"+
			"*Request ID:* `%s`\n\n"+
			"The SSM document will be created shortly.",
			request.Host,
			request.Port,
			getStringValue(request.SecurityApproverName),
			formatTimestamp(request.SecurityApprovalTimestamp),
			getStringValue(request.ManagerApproverName),
			formatTimestamp(request.ManagerApprovalTimestamp),
			request.RequestID,
		)
	} else if request.IsPartiallyApproved() {
		if request.HasSecurityApproval() {
			// Security approved, waiting for manager
			message = fmt.Sprintf("⏳ *Security Approval Granted*\n\n"+
				"Your request for access to `%s:%d` has received security approval.\n\n"+
				"*Security Approval:* %s at %s\n"+
				"*Waiting for:* Manager approval from %s\n"+
				"*Request ID:* `%s`",
				request.Host,
				request.Port,
				getStringValue(request.SecurityApproverName),
				formatTimestamp(request.SecurityApprovalTimestamp),
				request.ManagerGroupName,
				request.RequestID,
			)
		} else if request.HasManagerApproval() {
			// Manager approved, waiting for security
			message = fmt.Sprintf("⏳ *Manager Approval Granted*\n\n"+
				"Your request for access to `%s:%d` has received manager approval.\n\n"+
				"*Manager Approval:* %s at %s\n"+
				"*Waiting for:* Security approval\n"+
				"*Request ID:* `%s`",
				request.Host,
				request.Port,
				getStringValue(request.ManagerApproverName),
				formatTimestamp(request.ManagerApprovalTimestamp),
				request.RequestID,
			)
		}
	}
	
	if message != "" {
		_, _, err := n.client.PostMessage(ctx, request.UserID, slack.MsgOptionText(message, false))
		if err != nil {
			return fmt.Errorf("failed to send approval status update: %w", err)
		}
	}
	
	return nil
}

// getStringValue safely gets string value from pointer
func getStringValue(s *string) string {
	if s == nil {
		return "Unknown"
	}
	return *s
}

// formatTimestamp formats a timestamp pointer
func formatTimestamp(t *time.Time) string {
	if t == nil {
		return "Unknown"
	}
	return t.Format("2006-01-02 15:04 MST")
}

// UpdateApprovalMessages updates the approval request messages for all approvers
func (n *Notifier) UpdateApprovalMessages(ctx context.Context, request *models.AccessRequest, status string) error {
	if request.ApprovalMessageTimestamps == nil || len(request.ApprovalMessageTimestamps) == 0 {
		fmt.Printf("Warning: No message timestamps to update for request %s\n", request.RequestID)
		return nil
	}
	
	// Build updated message based on status
	var textMessage string
	var blocks []slack.Block
	
	switch status {
	case "fully_approved":
		textMessage = fmt.Sprintf("✅ *Request Fully Approved*\n\n"+
			"This request has been fully approved and is being processed.\n\n"+
			"*Request Details:*\n"+
			"• User: <@%s>\n"+
			"• Host: `%s:%d`\n"+
			"• Account: `%s`\n"+
			"• Expires: %s\n\n"+
			"*Approvals:*\n"+
			"• Security: %s\n"+
			"• Manager: %s\n\n"+
			"*Request ID:* `%s`",
			request.UserID,
			request.Host,
			request.Port,
			request.AccountID,
			request.ExpirationDate.Format("2006-01-02 15:04 MST"),
			getStringValue(request.SecurityApproverName),
			getStringValue(request.ManagerApproverName),
			request.RequestID)
		
		// No buttons for fully approved
		blocks = []slack.Block{
			slack.NewSectionBlock(
				&slack.TextBlockObject{
					Type: slack.MarkdownType,
					Text: textMessage,
				},
				nil,
				nil,
			),
		}
	
	case "security_approved":
		textMessage = fmt.Sprintf("⏳ *Partially Approved - Waiting for Manager*\n\n"+
			"Security approval has been granted. Waiting for manager approval.\n\n"+
			"*Request Details:*\n"+
			"• User: <@%s>\n"+
			"• Host: `%s:%d`\n"+
			"• Account: `%s`\n"+
			"• Expires: %s\n\n"+
			"*Approval Status:*\n"+
			"• ✅ Security: %s\n"+
			"• ⏳ Manager: Waiting for %s\n\n"+
			"*Request ID:* `%s`",
			request.UserID,
			request.Host,
			request.Port,
			request.AccountID,
			request.ExpirationDate.Format("2006-01-02 15:04 MST"),
			getStringValue(request.SecurityApproverName),
			request.ManagerGroupName,
			request.RequestID)
		
		// Show buttons for manager approval
		blocks = []slack.Block{
			slack.NewSectionBlock(
				&slack.TextBlockObject{
					Type: slack.MarkdownType,
					Text: textMessage,
				},
				nil,
				nil,
			),
			slack.NewActionBlock(
				"approval_actions",
				slack.NewButtonBlockElement(
					"approve",
					request.RequestID,
					&slack.TextBlockObject{Type: slack.PlainTextType, Text: "Approve"},
				).WithStyle(slack.StylePrimary),
				slack.NewButtonBlockElement(
					"deny",
					request.RequestID,
					&slack.TextBlockObject{Type: slack.PlainTextType, Text: "Deny"},
				).WithStyle(slack.StyleDanger),
			),
		}
	
	case "manager_approved":
		textMessage = fmt.Sprintf("⏳ *Partially Approved - Waiting for Security*\n\n"+
			"Manager approval has been granted. Waiting for security approval.\n\n"+
			"*Request Details:*\n"+
			"• User: <@%s>\n"+
			"• Host: `%s:%d`\n"+
			"• Account: `%s`\n"+
			"• Expires: %s\n\n"+
			"*Approval Status:*\n"+
			"• ⏳ Security: Waiting\n"+
			"• ✅ Manager: %s\n\n"+
			"*Request ID:* `%s`",
			request.UserID,
			request.Host,
			request.Port,
			request.AccountID,
			request.ExpirationDate.Format("2006-01-02 15:04 MST"),
			getStringValue(request.ManagerApproverName),
			request.RequestID)
		
		// Show buttons for security approval
		blocks = []slack.Block{
			slack.NewSectionBlock(
				&slack.TextBlockObject{
					Type: slack.MarkdownType,
					Text: textMessage,
				},
				nil,
				nil,
			),
			slack.NewActionBlock(
				"approval_actions",
				slack.NewButtonBlockElement(
					"approve",
					request.RequestID,
					&slack.TextBlockObject{Type: slack.PlainTextType, Text: "Approve"},
				).WithStyle(slack.StylePrimary),
				slack.NewButtonBlockElement(
					"deny",
					request.RequestID,
					&slack.TextBlockObject{Type: slack.PlainTextType, Text: "Deny"},
				).WithStyle(slack.StyleDanger),
			),
		}
	
	case "denied":
		deniedBy := getStringValue(request.Approver)
		reason := getStringValue(request.DenialReason)
		textMessage = fmt.Sprintf("❌ *Request Denied*\n\n"+
			"This request has been denied and will not be processed.\n\n"+
			"*Request Details:*\n"+
			"• User: <@%s>\n"+
			"• Host: `%s:%d`\n"+
			"• Account: `%s`\n\n"+
			"*Denial Information:*\n"+
			"• Denied by: %s\n"+
			"• Reason: %s\n\n"+
			"*Request ID:* `%s`",
			request.UserID,
			request.Host,
			request.Port,
			request.AccountID,
			deniedBy,
			reason,
			request.RequestID)
		
		// No buttons for denied
		blocks = []slack.Block{
			slack.NewSectionBlock(
				&slack.TextBlockObject{
					Type: slack.MarkdownType,
					Text: textMessage,
				},
				nil,
				nil,
			),
		}
	
	default:
		return fmt.Errorf("unknown status: %s", status)
	}
	
	// Update message for each approver
	successCount := 0
	for userID, msgRef := range request.ApprovalMessageTimestamps {
		// Parse msgRef to get channel and timestamp (format: "channel:timestamp")
		parts := strings.Split(msgRef, ":")
		if len(parts) != 2 {
			fmt.Printf("Warning: Invalid message reference format for %s: %s\n", userID, msgRef)
			continue
		}
		
		channel := parts[0]
		timestamp := parts[1]
		
		err := n.client.UpdateMessage(ctx, channel, timestamp, slack.MsgOptionBlocks(blocks...))
		if err != nil {
			fmt.Printf("Warning: Failed to update message for %s (channel: %s, timestamp: %s): %v\n", userID, channel, timestamp, err)
			// Continue to update other messages
		} else {
			successCount++
			fmt.Printf("Successfully updated message for %s (channel: %s, timestamp: %s)\n", userID, channel, timestamp)
		}
	}
	
	fmt.Printf("Updated %d/%d approval messages for request %s\n", successCount, len(request.ApprovalMessageTimestamps), request.RequestID)
	return nil
}
