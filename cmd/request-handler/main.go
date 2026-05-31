package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"

	"github.com/ssm-access-manager/internal/logging"
	"github.com/ssm-access-manager/internal/repository"
	"github.com/ssm-access-manager/internal/service"
	"github.com/ssm-access-manager/internal/slack"
	"github.com/ssm-access-manager/internal/validation"
	awshelper "github.com/ssm-access-manager/pkg/aws"
)

var (
	logger               *logging.Logger
	slackClient          *slack.Client
	slackNotifier        *slack.Notifier
	requestService       *service.AccessRequestService
	authService          *service.AuthorizationService
	approvalGroupService *service.ApprovalGroupService
	requestRepo          *repository.RequestRepository
	userRepo             *repository.UserRepository
	accountRepo          *repository.AccountRepository
	approvalGroupRepo    *repository.ApprovalGroupRepository
	validator            *validation.RequestValidator
	eventBridgeClient    *eventbridge.Client
	groupCache           *slack.GroupMembershipCache
)

func init() {
	var err error

	// Initialize logger
	logger, err = logging.NewProductionLogger()
	if err != nil {
		panic(fmt.Sprintf("failed to create logger: %v", err))
	}

	// Initialize Slack client
	slackBotToken := os.Getenv("SLACK_BOT_TOKEN")
	slackSigningSecret := os.Getenv("SLACK_SIGNING_SECRET")
	slackClient = slack.NewClient(slackBotToken, slackSigningSecret)
	slackNotifier = slack.NewNotifier(slackClient)

	// Initialize group membership cache with 5-minute TTL
	groupCache = slack.NewGroupMembershipCache(slackBotToken, 5*time.Minute)

	// Initialize AWS clients
	ctx := context.Background()
	roleAssumer, err := awshelper.NewRoleAssumer(ctx)
	if err != nil {
		logger.Fatal("failed to create role assumer")
	}

	dynamoClient := dynamodb.NewFromConfig(roleAssumer.BaseConfig)
	eventBridgeClient = eventbridge.NewFromConfig(roleAssumer.BaseConfig)

	// Initialize repositories
	requestsTable := os.Getenv("REQUESTS_TABLE")
	usersTable := os.Getenv("USERS_TABLE")
	accountsTable := os.Getenv("ACCOUNTS_TABLE")
	approvalGroupsTable := os.Getenv("APPROVAL_GROUPS_TABLE")

	requestRepo = repository.NewRequestRepository(dynamoClient, requestsTable)
	userRepo = repository.NewUserRepository(dynamoClient, usersTable)
	accountRepo = repository.NewAccountRepository(dynamoClient, accountsTable)
	approvalGroupRepo = repository.NewApprovalGroupRepository(dynamoClient, approvalGroupsTable)

	// Initialize services
	validator = validation.NewRequestValidator(90) // 90 days max expiration
	authService = service.NewAuthorizationService(userRepo, groupCache, nil)
	approvalGroupService = service.NewApprovalGroupService(approvalGroupRepo, authService, nil)
	requestService = service.NewAccessRequestService(requestRepo, validator, authService, nil)

	// Configure self-approval setting (for testing purposes only)
	// WARNING: Should only be enabled in test/development environments
	allowSelfApproval := os.Getenv("ALLOW_SELF_APPROVAL")
	if allowSelfApproval == "true" {
		logger.Warn("ALLOW_SELF_APPROVAL is enabled - this should only be used for testing!")
		requestService.SetAllowSelfApproval(true)
	}
}

// SlackCommand represents a Slack slash command payload
type SlackCommand struct {
	Token       string `json:"token"`
	TeamID      string `json:"team_id"`
	TeamDomain  string `json:"team_domain"`
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	UserID      string `json:"user_id"`
	UserName    string `json:"user_name"`
	Command     string `json:"command"`
	Text        string `json:"text"`
	ResponseURL string `json:"response_url"`
	TriggerID   string `json:"trigger_id"`
}

// InteractionPayload represents a Slack interaction payload
type InteractionPayload struct {
	Type        string              `json:"type"`
	User        InteractionUser     `json:"user"`
	View        InteractionView     `json:"view"`
	Actions     []InteractionAction `json:"actions"`
	TriggerID   string              `json:"trigger_id"`
	ResponseURL string              `json:"response_url"`
}

type InteractionUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

type InteractionView struct {
	ID         string               `json:"id"`
	CallbackID string               `json:"callback_id"`
	State      InteractionViewState `json:"state"`
}

type InteractionViewState struct {
	Values map[string]map[string]InteractionValue `json:"values"`
}

type InteractionValue struct {
	Type           string             `json:"type"`
	Value          string             `json:"value"`
	SelectedOption *InteractionOption `json:"selected_option"`
}

type InteractionOption struct {
	Text  InteractionText `json:"text"`
	Value string          `json:"value"`
}

type InteractionText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type InteractionAction struct {
	ActionID string `json:"action_id"`
	Value    string `json:"value"`
	Type     string `json:"type"`
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Log the incoming request for debugging
	contentType := request.Headers["content-type"]
	if contentType == "" {
		contentType = request.Headers["Content-Type"]
	}
	logger.Info(fmt.Sprintf("Received request: ContentType=%s, Body length=%d, Path=%s",
		contentType, len(request.Body), request.Path))

	// Log first 200 chars of body for debugging (be careful with sensitive data)
	bodyPreview := request.Body
	if len(bodyPreview) > 200 {
		bodyPreview = bodyPreview[:200] + "..."
	}
	logger.Info(fmt.Sprintf("Body preview: %s", bodyPreview))

	// Check for Slack URL verification challenge FIRST (before signature verification)
	// because verification requests have a different format
	if strings.Contains(request.Body, "\"type\":\"url_verification\"") {
		logger.Info("Received URL verification challenge")
		return handleURLVerification(request.Body)
	}

	// Verify Slack signature for all other requests
	headers := make(http.Header)
	for k, v := range request.Headers {
		headers.Set(k, v)
	}

	logger.Info(fmt.Sprintf("Verifying signature with headers: X-Slack-Request-Timestamp=%s, X-Slack-Signature=%s",
		request.Headers["x-slack-request-timestamp"],
		request.Headers["x-slack-signature"]))

	err := slackClient.VerifySignature(headers, request.Body)
	if err != nil {
		logger.Warn(fmt.Sprintf("Invalid Slack signature: %v", err))
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       "Unauthorized",
		}, nil
	}

	logger.Info("Signature verified successfully")

	// Log the request body type for debugging
	if strings.Contains(request.Body, "payload=") {
		logger.Info("Received interaction payload")
	}

	// Check if this is a modal submission (interaction)
	if isModalSubmission(request.Body) {
		return handleModalSubmission(ctx, request.Body)
	}

	// Check if this is a button interaction (approve/deny)
	if isButtonInteraction(request.Body) {
		return handleButtonInteraction(ctx, request.Body)
	}

	// Parse Slack command
	var cmd SlackCommand
	err = json.Unmarshal([]byte(request.Body), &cmd)
	if err != nil {
		// Try form-encoded format (this is what Slack actually sends)
		cmd, err = parseFormEncodedCommand(request.Body)
		if err != nil {
			logger.Error(fmt.Sprintf("failed to parse command: %v", err))
			return events.APIGatewayProxyResponse{
				StatusCode: 400,
				Body:       "Bad Request",
			}, nil
		}
	}

	logger.Info(fmt.Sprintf("Parsed command: UserID=%s, Text=%s", cmd.UserID, cmd.Text))

	// If no text provided, open modal
	if strings.TrimSpace(cmd.Text) == "" {
		return handleModalOpen(ctx, cmd)
	}

	// Otherwise, handle command-line format (backward compatibility)
	return handleCommandLineRequest(ctx, cmd)
}

// parseCommandText parses the command text into key-value pairs
func parseCommandText(text string) map[string]string {
	params := make(map[string]string)

	// Manual parsing to handle quoted values
	i := 0
	for i < len(text) {
		// Skip whitespace
		for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
			i++
		}
		if i >= len(text) {
			break
		}

		// Find key=value pair
		keyStart := i
		for i < len(text) && text[i] != '=' && text[i] != ' ' && text[i] != '\t' {
			i++
		}

		if i >= len(text) || text[i] != '=' {
			// No '=' found, skip this token
			for i < len(text) && text[i] != ' ' && text[i] != '\t' {
				i++
			}
			continue
		}

		key := text[keyStart:i]
		i++ // Skip '='

		if i >= len(text) {
			break
		}

		var value string

		// Check if value is quoted
		if text[i] == '"' {
			i++ // Skip opening quote
			valueStart := i
			// Find closing quote
			for i < len(text) && text[i] != '"' {
				if text[i] == '\\' && i+1 < len(text) {
					i++ // Skip escaped character
				}
				i++
			}
			value = text[valueStart:i]
			if i < len(text) {
				i++ // Skip closing quote
			}
		} else {
			// Unquoted value - read until whitespace
			valueStart := i

			// Check for Slack URL formatting
			if text[i] == '<' {
				// Find the closing '>'
				for i < len(text) && text[i] != '>' {
					i++
				}
				if i < len(text) {
					i++ // Include the '>'
				}
				rawValue := text[valueStart:i]

				// Strip Slack's URL formatting: <http://example.com|example.com> -> example.com
				// Also handles: <http://example.com> -> example.com
				if strings.HasPrefix(rawValue, "<") && strings.HasSuffix(rawValue, ">") {
					rawValue = strings.TrimPrefix(rawValue, "<")
					rawValue = strings.TrimSuffix(rawValue, ">")

					// If it contains a pipe, take the part after the pipe (the display text)
					if pipeIdx := strings.Index(rawValue, "|"); pipeIdx != -1 {
						value = rawValue[pipeIdx+1:]
					} else {
						// Otherwise, strip the protocol if present
						value = strings.TrimPrefix(rawValue, "http://")
						value = strings.TrimPrefix(value, "https://")
					}
				} else {
					value = rawValue
				}
			} else {
				// Regular unquoted value
				for i < len(text) && text[i] != ' ' && text[i] != '\t' {
					i++
				}
				value = text[valueStart:i]
			}
		}

		params[key] = value
	}

	return params
}

// parseFormEncodedCommand parses form-encoded Slack command
func parseFormEncodedCommand(body string) (SlackCommand, error) {
	var cmd SlackCommand

	// Parse URL-encoded form data
	values, err := url.ParseQuery(body)
	if err != nil {
		return cmd, fmt.Errorf("failed to parse form data: %w", err)
	}

	cmd.Token = values.Get("token")
	cmd.TeamID = values.Get("team_id")
	cmd.TeamDomain = values.Get("team_domain")
	cmd.ChannelID = values.Get("channel_id")
	cmd.ChannelName = values.Get("channel_name")
	cmd.UserID = values.Get("user_id")
	cmd.UserName = values.Get("user_name")
	cmd.Command = values.Get("command")
	cmd.Text = values.Get("text")
	cmd.ResponseURL = values.Get("response_url")
	cmd.TriggerID = values.Get("trigger_id")

	return cmd, nil
}

// isModalSubmission checks if the request body is a modal submission
func isModalSubmission(body string) bool {
	values, err := url.ParseQuery(body)
	if err != nil {
		return false
	}

	payload := values.Get("payload")
	if payload == "" {
		return false
	}

	var interaction InteractionPayload
	err = json.Unmarshal([]byte(payload), &interaction)
	if err != nil {
		return false
	}

	return interaction.Type == "view_submission" && interaction.View.CallbackID == "ssm_access_request"
}

// handleURLVerification handles Slack's URL verification challenge
func handleURLVerification(body string) (events.APIGatewayProxyResponse, error) {
	var challenge struct {
		Token     string `json:"token"`
		Challenge string `json:"challenge"`
		Type      string `json:"type"`
	}

	err := json.Unmarshal([]byte(body), &challenge)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Bad Request",
		}, nil
	}

	logger.Info(fmt.Sprintf("URL verification challenge received: %s", challenge.Challenge))

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "text/plain",
		},
		Body: challenge.Challenge,
	}, nil
}

// isButtonInteraction checks if the request body is a button interaction
func isButtonInteraction(body string) bool {
	values, err := url.ParseQuery(body)
	if err != nil {
		return false
	}

	payload := values.Get("payload")
	if payload == "" {
		return false
	}

	var interaction InteractionPayload
	err = json.Unmarshal([]byte(payload), &interaction)
	if err != nil {
		return false
	}

	return interaction.Type == "block_actions"
}

// handleButtonInteraction forwards button interactions to approval handler logic
func handleButtonInteraction(ctx context.Context, body string) (events.APIGatewayProxyResponse, error) {
	// Parse the interaction payload
	values, err := url.ParseQuery(body)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to parse interaction: %v", err))
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Bad Request",
		}, nil
	}

	payload := values.Get("payload")
	var interaction InteractionPayload
	err = json.Unmarshal([]byte(payload), &interaction)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to unmarshal interaction: %v", err))
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Bad Request",
		}, nil
	}

	// Get the action (approve or deny)
	if len(interaction.Actions) == 0 {
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       "",
		}, nil
	}

	action := interaction.Actions[0]
	requestID := action.Value

	logger.Info(fmt.Sprintf("Button interaction: action=%s, request_id=%s, user=%s, response_url=%s",
		action.ActionID, requestID, interaction.User.ID, interaction.ResponseURL))

	// Handle approve/deny
	switch action.ActionID {
	case "approve":
		logger.Info("Calling handleApproveButton")
		return handleApproveButton(ctx, interaction.User.ID, interaction.User.Username, requestID, interaction.ResponseURL)
	case "approve_security":
		logger.Info("Calling handleApproveSecurityButton")
		return handleApproveSecurityButton(ctx, interaction.User.ID, interaction.User.Username, requestID, interaction.ResponseURL)
	case "approve_manager":
		logger.Info("Calling handleApproveManagerButton")
		return handleApproveManagerButton(ctx, interaction.User.ID, interaction.User.Username, requestID, interaction.ResponseURL)
	case "deny":
		logger.Info("Calling handleDenyButton")
		return handleDenyButton(ctx, interaction.User.ID, interaction.User.Username, requestID, interaction.ResponseURL)
	default:
		logger.Warn(fmt.Sprintf("Unknown action: %s", action.ActionID))
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       "",
		}, nil
	}
}

// handleApproveButton handles the approve button click
func handleApproveButton(ctx context.Context, approverID, approverName, requestID, responseURL string) (events.APIGatewayProxyResponse, error) {
	// Get the request
	accessRequest, err := requestService.GetRequest(ctx, requestID)
	if err != nil {
		logger.LogError(ctx, "get_request", err, map[string]interface{}{
			"approver_id": approverID,
			"request_id":  requestID,
		})

		return sendEphemeralMessage(responseURL, fmt.Sprintf("❌ Failed to get request: %s", err.Error()))
	}

	// Check if user already provided an approval
	if accessRequest.SecurityApproverID != nil && *accessRequest.SecurityApproverID == approverID {
		return sendEphemeralMessage(responseURL, "✅ You already provided security approval for this request")
	}

	if accessRequest.ManagerApproverID != nil && *accessRequest.ManagerApproverID == approverID {
		return sendEphemeralMessage(responseURL, "✅ You already provided manager approval for this request")
	}

	// Get security group
	securityGroup, err := approvalGroupService.GetSecurityGroup(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to get security group: %v", err))
		return sendEphemeralMessage(responseURL, "❌ Failed to check authorization: security group not configured")
	}

	// Check if user is member of security group
	isSecurityMember, err := authService.IsGroupMember(ctx, securityGroup.GroupID, approverID)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to check security group membership: %v", err))
		isSecurityMember = false
	}
	logger.Info(fmt.Sprintf("Security group membership check: groupID=%s, userID=%s, isMember=%v", securityGroup.GroupID, approverID, isSecurityMember))

	// Check if user is member of request's manager group
	isManagerMember, err := authService.IsGroupMember(ctx, accessRequest.ManagerGroupID, approverID)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to check manager group membership: %v", err))
		isManagerMember = false
	}
	logger.Info(fmt.Sprintf("Manager group membership check: groupID=%s, userID=%s, isMember=%v", accessRequest.ManagerGroupID, approverID, isManagerMember))

	// If not member of any group, log unauthorized attempt and return error
	if !isSecurityMember && !isManagerMember {
		logger.Info("User is not member of any approval group - unauthorized")
		authService.LogUnauthorizedAttempt(ctx, approverID, approverName, requestID)
		return sendEphemeralMessage(responseURL, "❌ You are not authorized to approve this request")
	}

	// Check if the approval this user can provide is already granted by someone else
	if isSecurityMember && !isManagerMember && accessRequest.HasSecurityApproval() {
		// User is only in security group, but security approval already granted by someone else
		return sendEphemeralMessage(responseURL, fmt.Sprintf("✅ Security approval already granted by %s", *accessRequest.SecurityApproverName))
	}

	if isManagerMember && !isSecurityMember && accessRequest.HasManagerApproval() {
		// User is only in manager group, but manager approval already granted by someone else
		return sendEphemeralMessage(responseURL, fmt.Sprintf("✅ Manager approval already granted by %s", *accessRequest.ManagerApproverName))
	}

	// If member of BOTH groups, show approval type selection
	if isSecurityMember && isManagerMember {
		logger.Info("User is member of BOTH groups - showing approval type selection")
		needsSecurity := !accessRequest.HasSecurityApproval()
		needsManager := !accessRequest.HasManagerApproval()
		logger.Info(fmt.Sprintf("Approval status: needsSecurity=%v, needsManager=%v", needsSecurity, needsManager))

		// Check if both approvals already granted
		if !needsSecurity && !needsManager {
			return sendEphemeralMessage(responseURL, "✅ Both approvals already granted for this request")
		}

		return showApprovalTypeSelection(ctx, requestID, needsSecurity, needsManager, responseURL, securityGroup.GroupName, accessRequest.ManagerGroupName)
	}

	// If member of only security group, call ApproveRequestSecurity
	if isSecurityMember {
		logger.Info("User is member of security group only - calling handleApproveSecurityButton")
		return handleApproveSecurityButton(ctx, approverID, approverName, requestID, responseURL)
	}

	// If member of only manager group, call ApproveRequestManager
	if isManagerMember {
		logger.Info("User is member of manager group only - calling handleApproveManagerButton")
		return handleApproveManagerButton(ctx, approverID, approverName, requestID, responseURL)
	}

	// Should never reach here
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: `{"replace_original": false, "response_type": "ephemeral", "text": "❌ Unexpected error"}`,
	}, nil
}

// showApprovalTypeSelection shows buttons for user to choose approval type
func showApprovalTypeSelection(ctx context.Context, requestID string, needsSecurity, needsManager bool, responseURL, securityGroupName, managerGroupName string) (events.APIGatewayProxyResponse, error) {
	message := "You are a member of both approval groups. Please select which approval to provide:"

	// Build response structure
	type SlackButton struct {
		Type     string            `json:"type"`
		Text     map[string]string `json:"text"`
		ActionID string            `json:"action_id"`
		Value    string            `json:"value"`
		Style    string            `json:"style"`
	}

	type SlackBlock struct {
		Type     string            `json:"type"`
		Text     map[string]string `json:"text,omitempty"`
		Elements []SlackButton     `json:"elements,omitempty"`
	}

	type SlackResponse struct {
		ReplaceOriginal bool         `json:"replace_original"`
		ResponseType    string       `json:"response_type"`
		Text            string       `json:"text"`
		Blocks          []SlackBlock `json:"blocks"`
	}

	var buttons []SlackButton
	if needsSecurity {
		buttons = append(buttons, SlackButton{
			Type:     "button",
			Text:     map[string]string{"type": "plain_text", "text": fmt.Sprintf("Approve as %s", securityGroupName)},
			ActionID: "approve_security",
			Value:    requestID,
			Style:    "primary",
		})
	}
	if needsManager {
		buttons = append(buttons, SlackButton{
			Type:     "button",
			Text:     map[string]string{"type": "plain_text", "text": fmt.Sprintf("Approve as %s", managerGroupName)},
			ActionID: "approve_manager",
			Value:    requestID,
			Style:    "primary",
		})
	}

	// First, update the original message to remove buttons (replace with "Processing...")
	replaceResponse := SlackResponse{
		ReplaceOriginal: true,
		Text:            "⏳ Processing your approval...",
	}

	replaceBody, err := json.Marshal(replaceResponse)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to marshal replace response: %v", err))
	} else {
		// Post to response_url to replace original message
		resp, err := http.Post(responseURL, "application/json", strings.NewReader(string(replaceBody)))
		if err != nil {
			logger.Error(fmt.Sprintf("failed to post replace message: %v", err))
		} else {
			defer func() { _ = resp.Body.Close() }()
			logger.Info(fmt.Sprintf("Replaced original message, status: %d", resp.StatusCode))
		}
	}

	// Then, send ephemeral message with approval type selection
	response := SlackResponse{
		ReplaceOriginal: false,
		ResponseType:    "ephemeral",
		Text:            message,
		Blocks: []SlackBlock{
			{
				Type: "section",
				Text: map[string]string{"type": "mrkdwn", "text": message},
			},
			{
				Type:     "actions",
				Elements: buttons,
			},
		},
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to marshal approval type selection response: %v", err))
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: `{"text": "❌ Internal error"}`,
		}, nil
	}

	logger.Info(fmt.Sprintf("Posting approval type selection to response_url: %s", responseURL))

	// Post to response_url using HTTP client
	resp, err := http.Post(responseURL, "application/json", strings.NewReader(string(responseBody)))
	if err != nil {
		logger.Error(fmt.Sprintf("failed to post to response_url: %v", err))
	} else {
		defer func() { _ = resp.Body.Close() }()
		logger.Info(fmt.Sprintf("Posted to response_url, status: %d", resp.StatusCode))
	}

	// Return empty 200 response to acknowledge the button click
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "",
	}, nil
}

// handleApproveSecurityButton handles security approval
func handleApproveSecurityButton(ctx context.Context, approverID, approverName, requestID, responseURL string) (events.APIGatewayProxyResponse, error) {
	// Send immediate feedback via ephemeral message
	go func() {
		_, _ = sendEphemeralMessage(responseURL, "⏳ Processing your security approval...")
	}()

	// Call ApproveRequestSecurity service method
	accessRequest, err := requestService.ApproveRequestSecurity(ctx, requestID, approverID, approverName)
	if err != nil {
		logger.LogError(ctx, "approve_request_security", err, map[string]interface{}{
			"approver_id": approverID,
			"request_id":  requestID,
		})

		// Check if this is a self-approval error and display the specific message
		errorMsg := "❌ Failed to approve request"
		if err.Error() == "you cannot approve your own access request" {
			errorMsg = "❌ You cannot approve your own access request"
		}

		return sendEphemeralMessage(responseURL, errorMsg)
	}

	// Determine status for message updates
	var status string
	var confirmationMsg string

	if accessRequest.IsApproved() {
		status = "fully_approved"
		confirmationMsg = fmt.Sprintf("✅ *Security Approval Recorded*\n\n"+
			"Request is now *fully approved* (both security and manager approvals received).\n\n"+
			"*Request Details:*\n"+
			"• User: <@%s>\n"+
			"• Host: `%s:%d`\n"+
			"• Account: `%s`\n"+
			"• Expires: %s\n"+
			"• Request ID: `%s`",
			accessRequest.UserID,
			accessRequest.Host,
			accessRequest.Port,
			accessRequest.AccountID,
			accessRequest.ExpirationDate.Format("2006-01-02 15:04 MST"),
			requestID)

		// Publish event for document creation
		err = publishApprovalEvent(ctx, requestID)
		if err != nil {
			logger.Error(fmt.Sprintf("failed to publish approval event: %v", err))
		}

		// Send confirmation to requester
		err = slackNotifier.SendApprovalConfirmation(ctx, accessRequest.UserID, accessRequest)
		if err != nil {
			logger.Error(fmt.Sprintf("failed to send approval confirmation: %v", err))
		}
	} else {
		status = "security_approved"
		confirmationMsg = fmt.Sprintf("✅ *Security Approval Recorded*\n\n"+
			"Waiting for manager approval from %s.\n\n"+
			"*Request Details:*\n"+
			"• User: <@%s>\n"+
			"• Host: `%s:%d`\n"+
			"• Account: `%s`\n"+
			"• Expires: %s\n"+
			"• Request ID: `%s`",
			accessRequest.ManagerGroupName,
			accessRequest.UserID,
			accessRequest.Host,
			accessRequest.Port,
			accessRequest.AccountID,
			accessRequest.ExpirationDate.Format("2006-01-02 15:04 MST"),
			requestID)

		// Send status update to requester
		err = slackNotifier.SendApprovalStatusUpdate(ctx, accessRequest)
		if err != nil {
			logger.Error(fmt.Sprintf("failed to send approval status update: %v", err))
		}
	}

	// Update messages for all approvers
	err = slackNotifier.UpdateApprovalMessages(ctx, accessRequest, status)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to update approval messages: %v", err))
	}

	// Send success confirmation via ephemeral message
	return sendEphemeralMessage(responseURL, confirmationMsg)
}

// handleApproveManagerButton handles manager approval
func handleApproveManagerButton(ctx context.Context, approverID, approverName, requestID, responseURL string) (events.APIGatewayProxyResponse, error) {
	// Send immediate feedback via ephemeral message
	go func() {
		_, _ = sendEphemeralMessage(responseURL, "⏳ Processing your manager approval...")
	}()

	// Call ApproveRequestManager service method
	accessRequest, err := requestService.ApproveRequestManager(ctx, requestID, approverID, approverName)
	if err != nil {
		logger.LogError(ctx, "approve_request_manager", err, map[string]interface{}{
			"approver_id": approverID,
			"request_id":  requestID,
		})

		// Check if this is a self-approval error and display the specific message
		errorMsg := "❌ Failed to approve request"
		if err.Error() == "you cannot approve your own access request" {
			errorMsg = "❌ You cannot approve your own access request"
		}

		return sendEphemeralMessage(responseURL, errorMsg)
	}

	// Determine status for message updates
	var status string
	var confirmationMsg string

	if accessRequest.IsApproved() {
		status = "fully_approved"
		confirmationMsg = fmt.Sprintf("✅ *Manager Approval Recorded*\n\n"+
			"Request is now *fully approved* (both security and manager approvals received).\n\n"+
			"*Request Details:*\n"+
			"• User: <@%s>\n"+
			"• Host: `%s:%d`\n"+
			"• Account: `%s`\n"+
			"• Expires: %s\n"+
			"• Request ID: `%s`",
			accessRequest.UserID,
			accessRequest.Host,
			accessRequest.Port,
			accessRequest.AccountID,
			accessRequest.ExpirationDate.Format("2006-01-02 15:04 MST"),
			requestID)

		// Publish event for document creation
		err = publishApprovalEvent(ctx, requestID)
		if err != nil {
			logger.Error(fmt.Sprintf("failed to publish approval event: %v", err))
		}

		// Send confirmation to requester
		err = slackNotifier.SendApprovalConfirmation(ctx, accessRequest.UserID, accessRequest)
		if err != nil {
			logger.Error(fmt.Sprintf("failed to send approval confirmation: %v", err))
		}
	} else {
		status = "manager_approved"
		confirmationMsg = fmt.Sprintf("✅ *Manager Approval Recorded*\n\n"+
			"Waiting for security approval.\n\n"+
			"*Request Details:*\n"+
			"• User: <@%s>\n"+
			"• Host: `%s:%d`\n"+
			"• Account: `%s`\n"+
			"• Expires: %s\n"+
			"• Request ID: `%s`",
			accessRequest.UserID,
			accessRequest.Host,
			accessRequest.Port,
			accessRequest.AccountID,
			accessRequest.ExpirationDate.Format("2006-01-02 15:04 MST"),
			requestID)

		// Send status update to requester
		err = slackNotifier.SendApprovalStatusUpdate(ctx, accessRequest)
		if err != nil {
			logger.Error(fmt.Sprintf("failed to send approval status update: %v", err))
		}
	}

	// Update messages for all approvers
	err = slackNotifier.UpdateApprovalMessages(ctx, accessRequest, status)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to update approval messages: %v", err))
	}

	// Send success confirmation via ephemeral message
	return sendEphemeralMessage(responseURL, confirmationMsg)
}

// handleDenyButton handles the deny button click
func handleDenyButton(ctx context.Context, approverID, approverName, requestID, responseURL string) (events.APIGatewayProxyResponse, error) {
	logger.Info(fmt.Sprintf("handleDenyButton called: approverID=%s, requestID=%s", approverID, requestID))

	// Send immediate feedback via ephemeral message
	go func() {
		_, _ = sendEphemeralMessage(responseURL, "⏳ Processing denial...")
	}()

	// Get the request
	accessRequest, err := requestService.GetRequest(ctx, requestID)
	if err != nil {
		logger.LogError(ctx, "get_request", err, map[string]interface{}{
			"approver_id": approverID,
			"request_id":  requestID,
		})

		return sendEphemeralMessage(responseURL, fmt.Sprintf("❌ Failed to get request: %s", err.Error()))
	}

	// Check authorization - allow denial from either security or manager group members
	authorized := false

	// Get security group
	securityGroup, err := approvalGroupService.GetSecurityGroup(ctx)
	if err == nil {
		isSecurityMember, err := authService.IsGroupMember(ctx, securityGroup.GroupID, approverID)
		if err == nil && isSecurityMember {
			authorized = true
		}
	}

	// Check manager group membership
	if !authorized {
		isManagerMember, err := authService.IsGroupMember(ctx, accessRequest.ManagerGroupID, approverID)
		if err == nil && isManagerMember {
			authorized = true
		}
	}

	if !authorized {
		authService.LogUnauthorizedAttempt(ctx, approverID, approverName, requestID)
		return sendEphemeralMessage(responseURL, "❌ You are not authorized to deny this request")
	}

	// Deny the request with a default reason
	reason := "Denied via Slack button"

	logger.Info("Calling requestService.DenyRequest")
	accessRequest, err = requestService.DenyRequest(ctx, requestID, approverID, approverName, reason)
	if err != nil {
		logger.LogError(ctx, "deny_request", err, map[string]interface{}{
			"approver_id": approverID,
			"request_id":  requestID,
		})

		return sendEphemeralMessage(responseURL, fmt.Sprintf("❌ Failed to deny request: %s", err.Error()))
	}

	logger.Info("Request denied successfully, sending notification")

	// Send denial notification to requester
	err = slackNotifier.SendDenialNotification(ctx, accessRequest.UserID, accessRequest, reason)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to send denial notification: %v", err))
	}

	// Update messages for all approvers
	err = slackNotifier.UpdateApprovalMessages(ctx, accessRequest, "denied")
	if err != nil {
		logger.Error(fmt.Sprintf("failed to update approval messages: %v", err))
	}

	logger.Info("Returning success response")

	// Send detailed denial confirmation
	confirmationMsg := fmt.Sprintf("✅ *Request Denied*\n\n"+
		"*Request Details:*\n"+
		"• User: <@%s>\n"+
		"• Host: `%s:%d`\n"+
		"• Account: `%s`\n"+
		"• Request ID: `%s`\n\n"+
		"The requester has been notified.",
		accessRequest.UserID,
		accessRequest.Host,
		accessRequest.Port,
		accessRequest.AccountID,
		requestID)

	return sendEphemeralMessage(responseURL, confirmationMsg)
}

// publishApprovalEvent publishes an event to EventBridge for document creation
func publishApprovalEvent(ctx context.Context, requestID string) error {
	detailJSON, err := json.Marshal(map[string]string{
		"request_id": requestID,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal event detail: %w", err)
	}

	_, err = eventBridgeClient.PutEvents(ctx, &eventbridge.PutEventsInput{
		Entries: []types.PutEventsRequestEntry{
			{
				Source:     aws.String("ssm-access-manager"),
				DetailType: aws.String("Request Approved"),
				Detail:     aws.String(string(detailJSON)),
				Time:       aws.Time(time.Now()),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	logger.Info(fmt.Sprintf("Published approval event for request %s", requestID))
	return nil
}

// sendEphemeralMessage sends an ephemeral message via response_url
func sendEphemeralMessage(responseURL, message string) (events.APIGatewayProxyResponse, error) {
	payload := map[string]interface{}{
		"response_type": "ephemeral",
		"text":          message,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to marshal ephemeral message: %v", err))
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       "",
		}, nil
	}

	resp, err := http.Post(responseURL, "application/json", strings.NewReader(string(body)))
	if err != nil {
		logger.Error(fmt.Sprintf("failed to post ephemeral message: %v", err))
	} else {
		defer func() { _ = resp.Body.Close() }()
		logger.Info(fmt.Sprintf("Posted ephemeral message, status: %d", resp.StatusCode))
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "",
	}, nil
}

// handleModalOpen opens the access request modal
func handleModalOpen(ctx context.Context, cmd SlackCommand) (events.APIGatewayProxyResponse, error) {
	logger.Info(fmt.Sprintf("Opening modal for user %s, trigger_id: %s", cmd.UserID, cmd.TriggerID))

	// Get active accounts for dropdown
	accounts, err := accountRepo.ListActiveAccountsForDropdown(ctx)
	if err != nil {
		logger.LogError(ctx, "list_accounts", err, map[string]interface{}{})

		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: `{"response_type": "ephemeral", "text": "❌ Failed to load accounts. Please try again or use the command-line format:\n\n` + "`/ssm-access host=example.com port=8080 account=123456789012`" + `"}`,
		}, nil
	}

	logger.Info(fmt.Sprintf("Found %d accounts for dropdown", len(accounts)))

	if len(accounts) == 0 {
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: `{"response_type": "ephemeral", "text": "❌ No accounts configured. Please contact an administrator to add accounts."}`,
		}, nil
	}

	// Get active manager groups for dropdown
	managerGroups, err := approvalGroupService.ListActiveManagerGroups(ctx)
	if err != nil {
		logger.LogError(ctx, "list_manager_groups", err, map[string]interface{}{})

		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: `{"response_type": "ephemeral", "text": "❌ Failed to load manager groups. Please contact an administrator."}`,
		}, nil
	}

	logger.Info(fmt.Sprintf("Found %d manager groups for dropdown", len(managerGroups)))

	if len(managerGroups) == 0 {
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: `{"response_type": "ephemeral", "text": "❌ No manager groups configured. Please contact an administrator to configure approval groups."}`,
		}, nil
	}

	// Build and open modal
	logger.Info("Building modal view")
	view := slack.BuildAccessRequestModal(accounts, managerGroups)

	logger.Info("Calling Slack API to open view")
	resp, err := slackClient.OpenView(cmd.TriggerID, view)
	if err != nil {
		logger.LogError(ctx, "open_modal", err, map[string]interface{}{
			"user_id":    cmd.UserID,
			"trigger_id": cmd.TriggerID,
		})

		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: `{"response_type": "ephemeral", "text": "❌ Failed to open modal. Please use the command-line format:\n\n` + "`/ssm-access host=example.com port=8080 account=123456789012`" + `"}`,
		}, nil
	}

	logger.Info(fmt.Sprintf("Modal opened successfully, view_id: %s", resp.ID))

	// Return empty 200 response (modal is already open)
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "",
	}, nil
}

// handleModalSubmission processes modal form submission
func handleModalSubmission(ctx context.Context, body string) (events.APIGatewayProxyResponse, error) {
	// Parse the interaction payload
	values, err := url.ParseQuery(body)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to parse interaction: %v", err))
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Bad Request",
		}, nil
	}

	payload := values.Get("payload")
	var interaction InteractionPayload
	err = json.Unmarshal([]byte(payload), &interaction)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to unmarshal interaction: %v", err))
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Bad Request",
		}, nil
	}

	// Extract form values
	stateValues := interaction.View.State.Values

	accountID := ""
	if accountBlock, ok := stateValues["account_block"]; ok {
		if accountSelect, ok := accountBlock["account_select"]; ok && accountSelect.SelectedOption != nil {
			accountID = accountSelect.SelectedOption.Value
		}
	}

	host := ""
	if hostBlock, ok := stateValues["host_block"]; ok {
		if hostInput, ok := hostBlock["host_input"]; ok {
			host = hostInput.Value
		}
	}

	portStr := ""
	if portBlock, ok := stateValues["port_block"]; ok {
		if portInput, ok := portBlock["port_input"]; ok {
			portStr = portInput.Value
		}
	}

	expiresStr := ""
	if expiresBlock, ok := stateValues["expires_block"]; ok {
		if expiresInput, ok := expiresBlock["expires_input"]; ok {
			expiresStr = expiresInput.Value
		}
	}

	managerGroupID := ""
	managerGroupName := ""
	if managerGroupBlock, ok := stateValues["manager_group_block"]; ok {
		if managerGroupSelect, ok := managerGroupBlock["manager_group_select"]; ok && managerGroupSelect.SelectedOption != nil {
			managerGroupID = managerGroupSelect.SelectedOption.Value
			managerGroupName = managerGroupSelect.SelectedOption.Text.Text
		}
	}

	reason := ""
	if reasonBlock, ok := stateValues["reason_block"]; ok {
		if reasonInput, ok := reasonBlock["reason_input"]; ok {
			reason = reasonInput.Value
		}
	}

	// Convert port to int
	port, _ := strconv.Atoi(portStr)

	// Parse expiration date
	var expirationDate time.Time
	if expiresStr != "" {
		expirationDate, err = time.Parse("2006-01-02", expiresStr)
		if err != nil {
			// Return validation error
			return events.APIGatewayProxyResponse{
				StatusCode: 200,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: `{"response_action": "errors", "errors": {"expires_block": "Invalid date format. Use YYYY-MM-DD (e.g., 2026-03-15)"}}`,
			}, nil
		}
	} else {
		// Default to 14 days from now
		expirationDate = time.Now().AddDate(0, 0, 14)
	}

	// Validate parameters
	valid, missingFields := requestService.ValidateRequestParameters(host, port, accountID, expirationDate)
	if !valid {
		// Build validation errors for modal
		errors := make(map[string]string)
		for _, field := range missingFields {
			if strings.Contains(field, "host") {
				errors["host_block"] = field
			} else if strings.Contains(field, "port") {
				errors["port_block"] = field
			} else if strings.Contains(field, "account") {
				errors["account_block"] = field
			} else if strings.Contains(field, "expires") {
				errors["expires_block"] = field
			}
		}

		errorsJSON, _ := json.Marshal(map[string]interface{}{
			"response_action": "errors",
			"errors":          errors,
		})

		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: string(errorsJSON),
		}, nil
	}

	// Validate manager group selection
	if managerGroupID == "" || managerGroupName == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: `{"response_action": "errors", "errors": {"manager_group_block": "Manager group selection is required"}}`,
		}, nil
	}

	// Validate reason field
	if strings.TrimSpace(reason) == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: `{"response_action": "errors", "errors": {"reason_block": "Reason is required"}}`,
		}, nil
	}

	// Get username
	username := interaction.User.Username
	if username == "" {
		username = interaction.User.Name
	}
	if username == "" {
		username = interaction.User.ID
	}

	// Create access request
	accessRequest, err := requestService.CreateRequest(
		ctx,
		username,
		interaction.User.ID,
		host,
		port,
		accountID,
		expirationDate,
		managerGroupID,
		managerGroupName,
		reason,
	)
	if err != nil {
		logger.LogError(ctx, "create_request", err, map[string]interface{}{
			"user_id": interaction.User.ID,
			"host":    host,
			"port":    port,
		})

		// Return error in modal
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: fmt.Sprintf(`{"response_action": "errors", "errors": {"host_block": "Failed to create request: %s"}}`, err.Error()),
		}, nil
	}

	// Log the request
	logger.LogAccessRequest(ctx, accessRequest.RequestID, username, interaction.User.ID, host, port, accountID, expirationDate)

	// Send approval requests to groups
	securityGroup, err := approvalGroupService.GetSecurityGroup(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to get security group: %v", err))
	}

	managerGroup, err := approvalGroupService.GetGroup(ctx, accessRequest.ManagerGroupID)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to get manager group: %v", err))
	}

	if securityGroup != nil && managerGroup != nil {
		timestamps, err := slackNotifier.SendApprovalRequestToGroups(ctx, accessRequest, securityGroup, managerGroup, groupCache)
		if err != nil {
			logger.Error(fmt.Sprintf("failed to send approval request to groups: %v", err))
		}

		if len(timestamps) > 0 {
			// Store message timestamps for later updates
			accessRequest.ApprovalMessageTimestamps = timestamps
			err = requestRepo.SaveRequest(ctx, accessRequest)
			if err != nil {
				logger.Error(fmt.Sprintf("failed to save message timestamps: %v", err))
			} else {
				logger.Info(fmt.Sprintf("Stored %d message timestamps for request %s", len(timestamps), accessRequest.RequestID))
			}
		} else {
			logger.Error(fmt.Sprintf("WARNING: No approval messages were sent for request %s. Security group: %s (%s), Manager group: %s (%s)",
				accessRequest.RequestID,
				securityGroup.GroupName,
				securityGroup.GroupID,
				managerGroup.GroupName,
				managerGroup.GroupID))
		}
	} else {
		logger.Error(fmt.Sprintf("WARNING: Cannot send approval messages for request %s. Security group exists: %v, Manager group exists: %v",
			accessRequest.RequestID,
			securityGroup != nil,
			managerGroup != nil))
	}

	// Send confirmation to user
	_ = slackNotifier.SendRequestConfirmation(ctx, interaction.User.ID, accessRequest)

	// Return success (closes modal)
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: `{"response_action": "clear"}`,
	}, nil
}

// handleCommandLineRequest handles the legacy command-line format
func handleCommandLineRequest(ctx context.Context, cmd SlackCommand) (events.APIGatewayProxyResponse, error) {
	// Parse command parameters
	params := parseCommandText(cmd.Text)

	host := params["host"]
	portStr := params["port"]
	accountID := params["account"]
	expiresStr := params["expires"]
	reason := params["reason"]

	// Log parsed parameters for debugging
	logger.Info(fmt.Sprintf("Parsed parameters: host='%s' (len=%d), port='%s', account='%s', expires='%s', reason='%s'",
		host, len(host), portStr, accountID, expiresStr, reason))

	// Convert port to int
	port, _ := strconv.Atoi(portStr)

	// Parse expiration date
	var expirationDate time.Time
	var err error
	if expiresStr != "" {
		expirationDate, err = time.Parse("2006-01-02", expiresStr)
		if err != nil {
			logger.Error("invalid expiration date format")
			expirationDate = time.Time{}
		}
	} else {
		// Default to 14 days from now if not provided
		expirationDate = time.Now().AddDate(0, 0, 14)
		logger.Info("No expiration date provided, defaulting to 14 days from now")
	}

	// Validate parameters
	valid, missingFields := requestService.ValidateRequestParameters(host, port, accountID, expirationDate)

	// Add reason to validation
	if strings.TrimSpace(reason) == "" {
		valid = false
		missingFields = append(missingFields, "reason")
	}

	if !valid {
		logger.Info(fmt.Sprintf("Validation failed. Missing fields: %v", missingFields))

		// Return immediate response to Slack (must respond within 3 seconds)
		// Build error message
		fieldsText := ""
		for _, field := range missingFields {
			fieldsText += fmt.Sprintf("• %s\n", field)
		}

		responseText := fmt.Sprintf("❌ *Missing Required Fields*\n\n"+
			"Your access request is missing the following required fields:\n\n"+
			"%s\n"+
			"*Usage:* `/ssm-access host=example.com port=8080 account=123456789012 reason=\"Need to debug issue\" [expires=2025-12-31]`\n\n"+
			"*Note:* If expires is not provided, it defaults to 14 days from now.",
			fieldsText,
		)

		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: fmt.Sprintf(`{"response_type": "ephemeral", "text": %q}`, responseText),
		}, nil
	}

	// Use username from Slack command (no API call needed)
	username := cmd.UserName
	if username == "" {
		username = cmd.UserID // Fallback to user ID if username not provided
	}

	// Create access request (legacy format - no manager group)
	accessRequest, err := requestService.CreateRequest(
		ctx,
		username,
		cmd.UserID,
		host,
		port,
		accountID,
		expirationDate,
		"", // No manager group for legacy requests
		"", // No manager group name for legacy requests
		reason,
	)
	if err != nil {
		logger.LogError(ctx, "create_request", err, map[string]interface{}{
			"user_id": cmd.UserID,
			"host":    host,
			"port":    port,
		})

		errMsg := fmt.Sprintf("❌ Failed to create access request: %s", err.Error())

		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: fmt.Sprintf(`{"response_type": "ephemeral", "text": %q}`, errMsg),
		}, nil
	}

	// Log the request
	logger.LogAccessRequest(ctx, accessRequest.RequestID, username, cmd.UserID, host, port, accountID, expirationDate)

	// Get administrators and send approval request (legacy command-line format)
	admins, err := authService.GetAllAdministrators(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to get administrators: %v", err))

		// Still return success to user, but log the error
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: fmt.Sprintf(`{"response_type": "ephemeral", "text": "✅ Request submitted (ID: %s)\n\n⚠️ Warning: Could not notify administrators. Please contact an administrator."}`, accessRequest.RequestID),
		}, nil
	}

	// Send approval requests to administrators
	for _, admin := range admins {
		err = slackNotifier.SendApprovalRequest(ctx, admin.UserID, accessRequest)
		if err != nil {
			logger.Error(fmt.Sprintf("failed to send approval request to admin %s: %v", admin.UserID, err))
		}
	}

	// Return success message
	successMsg := fmt.Sprintf("✅ *Access Request Submitted*\n\n"+
		"Your request has been submitted for approval.\n\n"+
		"*Host:* `%s`\n"+
		"*Port:* `%d`\n"+
		"*Account:* `%s`\n"+
		"*Expires:* %s\n"+
		"*Request ID:* `%s`\n\n"+
		"You will be notified once a manager reviews your request.",
		host,
		port,
		accountID,
		expirationDate.Format("2006-01-02"),
		accessRequest.RequestID,
	)

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: fmt.Sprintf(`{"response_type": "ephemeral", "text": %q}`, successMsg),
	}, nil
}

func main() {
	lambda.Start(handler)
}
