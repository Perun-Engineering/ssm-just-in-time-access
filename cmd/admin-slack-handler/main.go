package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/ssm-access-manager/internal/audit"
	"github.com/ssm-access-manager/internal/logging"
	"github.com/ssm-access-manager/internal/models"
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
	authService          *service.AuthorizationService
	accountService       *service.AccountService
	requestService       *service.AccessRequestService
	documentService      *service.SSMDocumentService
	approvalGroupService *service.ApprovalGroupService
	auditService         *audit.AuditLogService
	userRepo             *repository.UserRepository
	requestRepo          *repository.RequestRepository
	documentRepo         *repository.DocumentRepository
	approvalGroupRepo    *repository.ApprovalGroupRepository
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

	// Initialize repositories
	usersTable := os.Getenv("USERS_TABLE")
	accountsTable := os.Getenv("ACCOUNTS_TABLE")
	requestsTable := os.Getenv("REQUESTS_TABLE")
	documentsTable := os.Getenv("DOCUMENTS_TABLE")
	approvalGroupsTable := os.Getenv("APPROVAL_GROUPS_TABLE")

	userRepo = repository.NewUserRepository(dynamoClient, usersTable)
	accountRepo := repository.NewAccountRepository(dynamoClient, accountsTable)
	requestRepo = repository.NewRequestRepository(dynamoClient, requestsTable)
	documentRepo = repository.NewDocumentRepository(dynamoClient, documentsTable)
	approvalGroupRepo = repository.NewApprovalGroupRepository(dynamoClient, approvalGroupsTable)

	// Initialize audit service
	auditLogGroup := os.Getenv("AUDIT_LOG_GROUP")
	if auditLogGroup == "" {
		auditLogGroup = "/aws/ssm-access-manager/audit"
	}
	slackTeamID := os.Getenv("SLACK_TEAM_ID")
	if slackTeamID == "" {
		slackTeamID = "unknown"
	}
	auditService = audit.NewAuditLogService(logger, auditLogGroup, slackTeamID)

	// Initialize services
	authService = service.NewAuthorizationService(userRepo, groupCache, auditService)
	approvalGroupService = service.NewApprovalGroupService(approvalGroupRepo, authService, auditService)
	accountService = service.NewAccountService(accountRepo, nil, roleAssumer, authService)
	requestService = service.NewAccessRequestService(requestRepo, nil, authService, auditService)

	// Initialize document service
	documentPrefix := os.Getenv("DOCUMENT_PREFIX")
	if documentPrefix == "" {
		documentPrefix = "PF" // Default: PortForwarding
	}
	nameGenerator := validation.NewDocumentNameGeneratorWithPrefix(documentPrefix)
	documentService = service.NewSSMDocumentService(documentRepo, roleAssumer, nameGenerator)

	// Wire up dependencies for revoke functionality
	requestService.SetDocumentRepository(documentRepo)
	requestService.SetDocumentService(documentService)
	requestService.SetSlackNotifier(slackNotifier)
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

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Log the incoming request
	logger.Info(fmt.Sprintf("Received admin command: ContentType=%s, Body length=%d",
		request.Headers["content-type"], len(request.Body)))

	// Parse Slack command
	var cmd SlackCommand
	err := json.Unmarshal([]byte(request.Body), &cmd)
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

	logger.Info(fmt.Sprintf("Parsed admin command: UserID=%s, Text=%s", cmd.UserID, cmd.Text))

	// Verify Slack signature
	headers := make(http.Header)
	for k, v := range request.Headers {
		headers.Set(k, v)
	}

	err = slackClient.VerifySignature(headers, request.Body)
	if err != nil {
		logger.Warn("invalid Slack signature")
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       "Unauthorized",
		}, nil
	}

	// Verify administrator authorization BEFORE processing any commands
	err = authService.VerifyAdministratorAuthorization(ctx, cmd.UserID)
	if err != nil {
		// Log unauthorized attempt
		authService.LogUnauthorizedAttempt(ctx, cmd.UserID, cmd.UserName, "admin_command")

		// Return empty response (no error message shown to user)
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       "",
		}, nil
	}

	// Parse command text - only split the first word (action), keep the rest intact
	cmd.Text = strings.TrimSpace(cmd.Text)
	spaceIndex := strings.Index(cmd.Text, " ")

	var action string
	var argsText string

	if spaceIndex == -1 {
		// No arguments, just the action
		action = cmd.Text
		argsText = ""
	} else {
		action = cmd.Text[:spaceIndex]
		argsText = strings.TrimSpace(cmd.Text[spaceIndex+1:])
	}

	if action == "" {
		return showHelp()
	}

	// Route to appropriate handler
	switch action {
	case "add-approval-group":
		return handleAddApprovalGroup(ctx, cmd.UserID, cmd.UserName, argsText)
	case "list-approval-groups":
		return handleListApprovalGroups(ctx)
	case "update-approval-group":
		return handleUpdateApprovalGroup(ctx, cmd.UserID, cmd.UserName, argsText)
	case "remove-approval-group":
		return handleRemoveApprovalGroup(ctx, cmd.UserID, cmd.UserName, argsText)
	case "audit-logs":
		return handleAuditLogs(ctx, argsText)
	case "add-admin":
		return handleAddAdmin(ctx, cmd.UserID, argsText)
	case "remove-admin":
		return handleRemoveAdmin(ctx, cmd.UserID, argsText)
	case "list-admins":
		return handleListAdmins(ctx)
	case "list-users":
		return handleListAllUsers(ctx)
	case "list-requests":
		return handleListRequests(ctx, argsText)
	case "approve-request":
		return handleApproveRequest(ctx, cmd.UserID, cmd.UserName, argsText)
	case "deny-request":
		return handleDenyRequest(ctx, cmd.UserID, cmd.UserName, argsText)
	case "cancel-request":
		return handleCancelRequest(ctx, cmd.UserID, argsText)
	case "revoke-request":
		return handleRevokeRequest(ctx, cmd.UserID, cmd.UserName, argsText)
	case "add-account":
		return handleAddAccount(ctx, cmd.UserID, argsText)
	case "update-account":
		return handleUpdateAccount(ctx, cmd.UserID, argsText)
	case "list-accounts":
		return handleListAccounts(ctx)
	case "help":
		return showHelp()
	default:
		return showHelp()
	}
}

func handleAddAdmin(ctx context.Context, adminID string, argsText string) (events.APIGatewayProxyResponse, error) {
	args := strings.Fields(argsText)
	if len(args) < 1 {
		return slackResponse("❌ *Missing Arguments*\n\nUsage: `/ssm-admin add-admin @user`\n\nExample: `/ssm-admin add-admin @jane.admin`")
	}

	// Extract user ID from mention
	userMention := args[0]
	userID := extractUserID(userMention)

	if userID == "" {
		return slackResponse("❌ *Invalid User*\n\nPlease mention a user using @username\n\nExample: `/ssm-admin add-admin @jane.admin`")
	}

	// Get user info from Slack
	username := userID
	email := fmt.Sprintf("%s@slack.local", userID)

	user, err := slackClient.GetUserInfo(ctx, userID)
	if err == nil {
		if user.RealName != "" {
			username = user.RealName
		} else if user.Name != "" {
			username = user.Name
		}
		if user.Profile.Email != "" {
			email = user.Profile.Email
		}
	}

	// Add administrator
	err = authService.AddAdministrator(ctx, userID, username, email, adminID)
	if err != nil {
		logger.LogError(ctx, "add_administrator", err, map[string]interface{}{
			"admin_id": adminID,
			"user_id":  userID,
		})
		return slackResponse(fmt.Sprintf("❌ *Failed to Add Administrator*\n\n%s", err.Error()))
	}

	return slackResponse(fmt.Sprintf("✅ *Administrator Added*\n\n<@%s> is now an administrator and can manage users and accounts.", userID))
}

func handleRemoveAdmin(ctx context.Context, adminID string, argsText string) (events.APIGatewayProxyResponse, error) {
	args := strings.Fields(argsText)
	if len(args) < 1 {
		return slackResponse("❌ *Missing Arguments*\n\nUsage: `/ssm-admin remove-admin @user`\n\nExample: `/ssm-admin remove-admin @jane.admin`")
	}

	userMention := args[0]
	userID := extractUserID(userMention)

	if userID == "" {
		return slackResponse("❌ *Invalid User*\n\nPlease mention a user using @username")
	}

	err := authService.RemoveAdministrator(ctx, userID, adminID)
	if err != nil {
		logger.LogError(ctx, "remove_administrator", err, map[string]interface{}{
			"admin_id": adminID,
			"user_id":  userID,
		})
		return slackResponse(fmt.Sprintf("❌ *Failed to Remove Administrator*\n\n%s", err.Error()))
	}

	return slackResponse(fmt.Sprintf("✅ *Administrator Removed*\n\n<@%s> is no longer an administrator.", userID))
}

func handleListAdmins(ctx context.Context) (events.APIGatewayProxyResponse, error) {
	admins, err := authService.GetAllAdministrators(ctx)
	if err != nil {
		logger.LogError(ctx, "list_administrators", err, map[string]interface{}{})
		return slackResponse("❌ *Failed to List Administrators*\n\nPlease try again later.")
	}

	if len(admins) == 0 {
		return slackResponse("📋 *Administrators*\n\nNo administrators found.")
	}

	message := "📋 *Administrators*\n\n"
	for _, admin := range admins {
		message += fmt.Sprintf("• <@%s> (%s)\n", admin.UserID, admin.Email)
	}

	return slackResponse(message)
}

func handleListAllUsers(ctx context.Context) (events.APIGatewayProxyResponse, error) {
	// Note: Manager role has been deprecated - only list administrators now
	admins, err := authService.GetAllAdministrators(ctx)
	if err != nil {
		return slackResponse("❌ *Failed to List Users*\n\nPlease try again later.")
	}

	message := "📋 *All Users*\n\n"

	if len(admins) > 0 {
		message += "*Administrators:*\n"
		for _, admin := range admins {
			message += fmt.Sprintf("• <@%s> (%s)\n", admin.UserID, admin.Email)
		}
		message += "\n"
	}

	if len(admins) == 0 {
		message += "No users found."
	}

	message += "\n*Note:* Manager role has been replaced with approval groups. Use `/ssm-admin list-approval-groups` to see approval groups."

	return slackResponse(message)
}

func showHelp() (events.APIGatewayProxyResponse, error) {
	help := "📚 *SSM Admin Commands*\n\n" +
		"*Approval Group Management:*\n" +
		"• `/ssm-admin add-approval-group group_id=<id> name=<name> type=<security|manager>` - Add approval group\n" +
		"• `/ssm-admin list-approval-groups` - List all approval groups\n" +
		"• `/ssm-admin update-approval-group group_id=<id> [name=<name>] [active=<true|false>]` - Update group\n" +
		"• `/ssm-admin remove-approval-group <group_id>` - Remove approval group\n\n" +
		"*User Management:*\n" +
		"• `/ssm-admin add-admin @user` - Add an administrator\n" +
		"• `/ssm-admin remove-admin @user` - Remove an administrator\n" +
		"• `/ssm-admin list-admins` - List all administrators\n" +
		"• `/ssm-admin list-users` - List all users\n\n" +
		"*Request Management:*\n" +
		"• `/ssm-admin list-requests [pending|active|all]` - List access requests (managers+)\n" +
		"• `/ssm-admin approve-request <id>` - Approve a request (managers+)\n" +
		"• `/ssm-admin deny-request <id> <reason>` - Deny a request (admins only)\n" +
		"• `/ssm-admin cancel-request <id>` - Cancel any request (admins only)\n" +
		"• `/ssm-admin revoke-request <id> [reason=\"reason\"]` - Revoke approved request (admins only)\n\n" +
		"*Audit Logs:*\n" +
		"• `/ssm-admin audit-logs [request_id=<id>] [user_id=<id>] [event_type=<type>]` - Generate audit log query\n\n" +
		"*Account Management:*\n" +
		"• `/ssm-admin add-account account_id=<id> account_name=<name> role_name=<role> regions=<regions> [bastion_host_id=<id>]` - Add account\n" +
		"• `/ssm-admin update-account account_id=<id> account_name=<name> role_name=<role> regions=<regions> [bastion_host_id=<id>]` - Update account\n" +
		"• `/ssm-admin list-accounts` - List all accounts\n\n" +
		"*Roles:*\n" +
		"• *Administrator* - Can manage users, accounts, and approval groups\n\n" +
		"*Examples:*\n" +
		"`/ssm-admin add-approval-group group_id=S12345678 name=\"Security Team\" type=security`\n" +
		"`/ssm-admin list-approval-groups`\n" +
		"`/ssm-admin audit-logs request_id=c75bcdd5-cd44-4048-9c6a-b42e18b8451f`\n" +
		"`/ssm-admin add-admin @john.doe`\n" +
		"`/ssm-admin list-requests`\n" +
		"`/ssm-admin approve-request c75bcdd5-cd44-4048-9c6a-b42e18b8451f`\n" +
		"`/ssm-admin revoke-request c75bcdd5-cd44-4048-9c6a-b42e18b8451f reason=\"Security incident\"`\n" +
		"`/ssm-admin add-account account_id=123456789012 account_name=\"Production\" role_name=SSMDocumentManagerRole regions=us-east-1 bastion_host_id=i-1234567890abcdef0`\n\n" +
		"*Note:* Most commands require administrator role. Approval groups replace the old manager role system."

	return slackResponse(help)
}

// extractUserID extracts user ID from Slack mention format
// Formats: <@U12345678|username> or <@U12345678>
func extractUserID(mention string) string {
	mention = strings.TrimSpace(mention)

	// Remove < and >
	mention = strings.Trim(mention, "<>")

	// Remove @ prefix
	mention = strings.TrimPrefix(mention, "@")

	// Split by | to handle <@U12345678|username> format
	parts := strings.Split(mention, "|")
	userID := parts[0]

	// Validate it looks like a Slack user ID
	if strings.HasPrefix(userID, "U") && len(userID) > 5 {
		return userID
	}

	return ""
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

// slackResponse creates a Slack ephemeral response
func slackResponse(text string) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: fmt.Sprintf(`{"response_type": "ephemeral", "text": %q}`, text),
	}, nil
}

// handleListRequests lists access requests
func handleListRequests(ctx context.Context, argsText string) (events.APIGatewayProxyResponse, error) {
	// Default to pending requests
	status := "pending"
	if argsText != "" {
		status = strings.TrimSpace(argsText)
	}

	var requests []*models.AccessRequest
	var err error

	switch status {
	case "pending":
		requests, err = requestService.ListPendingRequests(ctx)
	case "active":
		// Get all requests and filter for approved + not expired
		requests, err = requestService.ListAllRequests(ctx)
		if err == nil {
			now := time.Now()
			var activeRequests []*models.AccessRequest
			for _, req := range requests {
				if req.IsApproved() && req.ExpirationDate.After(now) {
					activeRequests = append(activeRequests, req)
				}
			}
			requests = activeRequests
		}
	case "all":
		requests, err = requestService.ListAllRequests(ctx)
	default:
		return slackResponse("❌ *Invalid Status*\n\nValid options: `pending`, `active`, `all`\n\nUsage: `/ssm-admin list-requests [pending|active|all]`")
	}

	if err != nil {
		logger.LogError(ctx, "list_requests", err, map[string]interface{}{})
		return slackResponse("❌ *Failed to List Requests*\n\nPlease try again later.")
	}

	// For "active" status, requests are already filtered
	// For "pending" and "all", filter out expired requests
	if status != "all" {
		now := time.Now()
		var filteredRequests []*models.AccessRequest
		for _, req := range requests {
			if req.ExpirationDate.After(now) {
				filteredRequests = append(filteredRequests, req)
			}
		}
		requests = filteredRequests
	}

	if len(requests) == 0 {
		return slackResponse(fmt.Sprintf("📋 *Access Requests (%s)*\n\nNo %s requests found.", status, status))
	}

	message := fmt.Sprintf("📋 *Access Requests (%s)*\n\n", status)
	for i, req := range requests {
		if i >= 10 {
			message += fmt.Sprintf("\n_...and %d more requests_", len(requests)-10)
			break
		}
		message += fmt.Sprintf("*Request ID:* `%s`\n", req.RequestID)
		message += fmt.Sprintf("• User: <@%s>\n", req.UserID)
		message += fmt.Sprintf("• Host: `%s:%d`\n", req.Host, req.Port)
		message += fmt.Sprintf("• Account: `%s`\n", req.AccountID)
		message += fmt.Sprintf("• Expires: %s\n", req.ExpirationDate.Format("2006-01-02"))
		message += fmt.Sprintf("• Status: %s\n\n", req.Status)
	}

	message += "\n*Commands:*\n"
	message += "• `/ssm-admin approve-request <request_id>`\n"
	message += "• `/ssm-admin deny-request <request_id> <reason>`\n"
	message += "• `/ssm-admin cancel-request <request_id>` (admin only)\n"
	message += "• `/ssm-admin revoke-request <request_id> [reason=\"reason\"]` (admin only)"

	return slackResponse(message)
}

// handleApproveRequest approves a pending request
func handleApproveRequest(ctx context.Context, approverID, approverName string, argsText string) (events.APIGatewayProxyResponse, error) {
	args := strings.Fields(argsText)
	if len(args) < 1 {
		return slackResponse("❌ *Missing Arguments*\n\nUsage: `/ssm-admin approve-request <request_id>`\n\nExample: `/ssm-admin approve-request c75bcdd5-cd44-4048-9c6a-b42e18b8451f`")
	}

	requestID := args[0]

	// Approve the request
	accessRequest, err := requestService.ApproveRequest(ctx, requestID, approverID, approverName)
	if err != nil {
		logger.LogError(ctx, "approve_request", err, map[string]interface{}{
			"approver_id": approverID,
			"request_id":  requestID,
		})
		return slackResponse(fmt.Sprintf("❌ *Failed to Approve Request*\n\n%s", err.Error()))
	}

	// Notify the requester
	err = slackNotifier.SendApprovalConfirmation(ctx, accessRequest.UserID, accessRequest)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to send approval confirmation: %v", err))
	}

	return slackResponse(fmt.Sprintf("✅ *Request Approved*\n\n"+
		"Request `%s` has been approved.\n\n"+
		"• User: <@%s>\n"+
		"• Host: `%s:%d`\n"+
		"• Account: `%s`\n\n"+
		"The SSM document will be created shortly.",
		requestID,
		accessRequest.UserID,
		accessRequest.Host,
		accessRequest.Port,
		accessRequest.AccountID,
	))
}

// handleDenyRequest denies a pending request
func handleDenyRequest(ctx context.Context, approverID, approverName string, argsText string) (events.APIGatewayProxyResponse, error) {
	args := strings.Fields(argsText)
	if len(args) < 2 {
		return slackResponse("❌ *Missing Arguments*\n\nUsage: `/ssm-admin deny-request <request_id> <reason>`\n\nExample: `/ssm-admin deny-request c75bcdd5... \"Insufficient justification\"`")
	}

	requestID := args[0]
	reason := strings.Join(args[1:], " ")

	// Deny the request
	accessRequest, err := requestService.DenyRequest(ctx, requestID, approverID, approverName, reason)
	if err != nil {
		logger.LogError(ctx, "deny_request", err, map[string]interface{}{
			"approver_id": approverID,
			"request_id":  requestID,
		})
		return slackResponse(fmt.Sprintf("❌ *Failed to Deny Request*\n\n%s", err.Error()))
	}

	// Notify the requester
	err = slackNotifier.SendDenialNotification(ctx, accessRequest.UserID, accessRequest, reason)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to send denial notification: %v", err))
	}

	return slackResponse(fmt.Sprintf("✅ *Request Denied*\n\n"+
		"Request `%s` has been denied.\n\n"+
		"• User: <@%s>\n"+
		"• Reason: %s",
		requestID,
		accessRequest.UserID,
		reason,
	))
}

// handleCancelRequest cancels any request (admin only)
func handleCancelRequest(ctx context.Context, adminID string, argsText string) (events.APIGatewayProxyResponse, error) {
	args := strings.Fields(argsText)
	if len(args) < 1 {
		return slackResponse("❌ *Missing Arguments*\n\nUsage: `/ssm-admin cancel-request <request_id>`\n\nExample: `/ssm-admin cancel-request c75bcdd5-cd44-4048-9c6a-b42e18b8451f`")
	}

	requestID := args[0]

	// Get the request
	accessRequest, err := requestService.GetRequest(ctx, requestID)
	if err != nil {
		logger.LogError(ctx, "get_request", err, map[string]interface{}{
			"admin_id":   adminID,
			"request_id": requestID,
		})
		return slackResponse(fmt.Sprintf("❌ *Failed to Get Request*\n\n%s", err.Error()))
	}

	// Cancel by denying with admin reason
	_, err = requestService.DenyRequest(ctx, requestID, adminID, "Administrator", "Cancelled by administrator")
	if err != nil {
		logger.LogError(ctx, "cancel_request", err, map[string]interface{}{
			"admin_id":   adminID,
			"request_id": requestID,
		})
		return slackResponse(fmt.Sprintf("❌ *Failed to Cancel Request*\n\n%s", err.Error()))
	}

	return slackResponse(fmt.Sprintf("✅ *Request Cancelled*\n\n"+
		"Request `%s` has been cancelled.\n\n"+
		"• User: <@%s>\n"+
		"• Host: `%s:%d`\n"+
		"• Status was: %s",
		requestID,
		accessRequest.UserID,
		accessRequest.Host,
		accessRequest.Port,
		accessRequest.Status,
	))
}

// handleAddAccount adds a new AWS account
func handleAddAccount(ctx context.Context, adminID string, argsText string) (events.APIGatewayProxyResponse, error) {
	// Parse key=value arguments directly from the text
	params := parseKeyValueArgs([]string{argsText})

	// Debug logging
	logger.Info(fmt.Sprintf("Add account - parsed params: %+v", params))
	logger.Info(fmt.Sprintf("Add account - raw text: %s", argsText))

	accountID := params["account_id"]
	accountName := params["account_name"]
	roleName := params["role_name"]
	regionsStr := params["regions"]
	bastionHostID := params["bastion_host_id"] // Optional

	// Validate required parameters
	if accountID == "" {
		return slackResponse("❌ *Missing Parameter*\n\nRequired: `account_id`\n\nUsage: `/ssm-admin add-account account_id=<id> account_name=<name> role_name=<role> regions=<regions> [bastion_host_id=<id>]`")
	}
	if accountName == "" {
		return slackResponse("❌ *Missing Parameter*\n\nRequired: `account_name`\n\nUsage: `/ssm-admin add-account account_id=<id> account_name=<name> role_name=<role> regions=<regions> [bastion_host_id=<id>]`")
	}
	if roleName == "" {
		return slackResponse("❌ *Missing Parameter*\n\nRequired: `role_name`\n\nUsage: `/ssm-admin add-account account_id=<id> account_name=<name> role_name=<role> regions=<regions> [bastion_host_id=<id>]`")
	}
	if regionsStr == "" {
		return slackResponse("❌ *Missing Parameter*\n\nRequired: `regions`\n\nUsage: `/ssm-admin add-account account_id=<id> account_name=<name> role_name=<role> regions=<regions> [bastion_host_id=<id>]`")
	}

	// Parse regions (comma-separated)
	regions := strings.Split(regionsStr, ",")
	for i := range regions {
		regions[i] = strings.TrimSpace(regions[i])
	}

	// Add account
	account, err := accountService.AddAccount(ctx, accountID, accountName, roleName, regions, bastionHostID, adminID)
	if err != nil {
		logger.LogError(ctx, "add_account", err, map[string]interface{}{
			"admin_id":   adminID,
			"account_id": accountID,
		})
		return slackResponse(fmt.Sprintf("❌ *Failed to Add Account*\n\n%s", err.Error()))
	}

	message := fmt.Sprintf("✅ *Account Added*\n\n"+
		"*Account ID:* `%s`\n"+
		"*Account Name:* %s\n"+
		"*Role Name:* %s\n"+
		"*Regions:* %s\n",
		account.AccountID,
		account.AccountName,
		account.RoleName,
		strings.Join(account.Regions, ", "),
	)

	if account.BastionHostID != "" {
		message += fmt.Sprintf("*Bastion Host ID:* `%s`\n", account.BastionHostID)
	}

	return slackResponse(message)
}

// handleUpdateAccount updates an existing AWS account
func handleUpdateAccount(ctx context.Context, adminID string, argsText string) (events.APIGatewayProxyResponse, error) {
	// Debug logging - show raw text with character codes
	logger.Info(fmt.Sprintf("Update account - raw text: %q", argsText))
	logger.Info(fmt.Sprintf("Update account - raw text bytes: %v", []byte(argsText)))

	// Parse key=value arguments directly from the text
	params := parseKeyValueArgs([]string{argsText})

	// Debug logging
	logger.Info(fmt.Sprintf("Update account - parsed params: %+v", params))

	accountID := params["account_id"]
	accountName := params["account_name"]
	roleName := params["role_name"]
	regionsStr := params["regions"]
	bastionHostID := params["bastion_host_id"] // Optional

	// Validate required parameters
	if accountID == "" {
		return slackResponse("❌ *Missing Parameter*\n\nRequired: `account_id`\n\nUsage: `/ssm-admin update-account account_id=<id> account_name=<name> role_name=<role> regions=<regions> [bastion_host_id=<id>]`")
	}
	if accountName == "" {
		return slackResponse("❌ *Missing Parameter*\n\nRequired: `account_name`\n\nUsage: `/ssm-admin update-account account_id=<id> account_name=<name> role_name=<role> regions=<regions> [bastion_host_id=<id>]`")
	}
	if roleName == "" {
		return slackResponse("❌ *Missing Parameter*\n\nRequired: `role_name`\n\nUsage: `/ssm-admin update-account account_id=<id> account_name=<name> role_name=<role> regions=<regions> [bastion_host_id=<id>]`")
	}
	if regionsStr == "" {
		return slackResponse("❌ *Missing Parameter*\n\nRequired: `regions`\n\nUsage: `/ssm-admin update-account account_id=<id> account_name=<name> role_name=<role> regions=<regions> [bastion_host_id=<id>]`")
	}

	// Parse regions (comma-separated)
	regions := strings.Split(regionsStr, ",")
	for i := range regions {
		regions[i] = strings.TrimSpace(regions[i])
	}

	// Update account
	account, err := accountService.UpdateAccount(ctx, accountID, accountName, roleName, regions, bastionHostID, adminID)
	if err != nil {
		logger.LogError(ctx, "update_account", err, map[string]interface{}{
			"admin_id":   adminID,
			"account_id": accountID,
		})
		return slackResponse(fmt.Sprintf("❌ *Failed to Update Account*\n\n%s", err.Error()))
	}

	message := fmt.Sprintf("✅ *Account Updated*\n\n"+
		"*Account ID:* `%s`\n"+
		"*Account Name:* %s\n"+
		"*Role Name:* %s\n"+
		"*Regions:* %s\n",
		account.AccountID,
		account.AccountName,
		account.RoleName,
		strings.Join(account.Regions, ", "),
	)

	if account.BastionHostID != "" {
		message += fmt.Sprintf("*Bastion Host ID:* `%s`\n", account.BastionHostID)
	}

	return slackResponse(message)
}

// handleListAccounts lists all accounts
func handleListAccounts(ctx context.Context) (events.APIGatewayProxyResponse, error) {
	accounts, err := accountService.ListAccounts(ctx)
	if err != nil {
		logger.LogError(ctx, "list_accounts", err, map[string]interface{}{})
		return slackResponse("❌ *Failed to List Accounts*\n\nPlease try again later.")
	}

	if len(accounts) == 0 {
		return slackResponse("📋 *AWS Accounts*\n\nNo accounts configured.")
	}

	message := "📋 *AWS Accounts*\n\n"
	for _, account := range accounts {
		message += fmt.Sprintf("*%s* (`%s`)\n", account.AccountName, account.AccountID)
		message += fmt.Sprintf("• Role: %s\n", account.RoleName)
		message += fmt.Sprintf("• Regions: %s\n", strings.Join(account.Regions, ", "))
		if account.BastionHostID != "" {
			message += fmt.Sprintf("• Bastion Host: `%s`\n", account.BastionHostID)
		}
		message += fmt.Sprintf("• Status: %s\n\n", account.Status)
	}

	return slackResponse(message)
}

// parseKeyValueArgs parses key=value arguments from command text
func parseKeyValueArgs(args []string) map[string]string {
	params := make(map[string]string)

	// Join all args back together
	text := strings.Join(args, " ")

	// Split by spaces, but respect quotes (both ASCII and Unicode smart quotes)
	var tokens []string
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0)

	for _, ch := range text {
		// Handle both ASCII quotes and Unicode smart quotes
		isOpenQuote := ch == '"' || ch == '\'' || ch == '\u201C' || ch == '\u2018' || ch == '\u2039' || ch == '\u00AB'
		isCloseQuote := ch == '"' || ch == '\'' || ch == '\u201D' || ch == '\u2019' || ch == '\u203A' || ch == '\u00BB'

		if isOpenQuote && !inQuotes {
			inQuotes = true
			quoteChar = ch
			continue
		}

		// For smart quotes, match opening with closing
		if inQuotes {
			if (quoteChar == '"' && ch == '"') || // ASCII double quote
				(quoteChar == '\'' && ch == '\'') || // ASCII single quote
				(quoteChar == '\u201C' && ch == '\u201D') || // Smart double quotes
				(quoteChar == '\u2018' && ch == '\u2019') || // Smart single quotes
				(quoteChar == '\u2039' && ch == '\u203A') || // Single angle quotes
				(quoteChar == '\u00AB' && ch == '\u00BB') || // Double angle quotes
				isCloseQuote { // Any closing quote
				inQuotes = false
				quoteChar = 0
				continue
			}
		}

		if ch == ' ' && !inQuotes {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteRune(ch)
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	// Debug: log tokens
	fmt.Printf("DEBUG: Tokens: %v\n", tokens)

	// Now parse each token as key=value
	for _, token := range tokens {
		parts := strings.SplitN(token, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// Remove any remaining quotes from the value
			value = strings.Trim(value, "\"'\u201C\u201D\u2018\u2019\u00AB\u00BB\u2039\u203A")
			params[key] = value
			fmt.Printf("DEBUG: Parsed %s = %s\n", key, value)
		}
	}

	return params
}

// handleRevokeRequest revokes an approved access request
func handleRevokeRequest(ctx context.Context, adminID, adminName string, argsText string) (events.APIGatewayProxyResponse, error) {
	args := strings.Fields(argsText)
	if len(args) < 1 {
		return slackResponse("❌ *Missing Request ID*\n\nUsage: `/ssm-admin revoke-request <request_id> [reason=\"reason\"]`\n\nExample: `/ssm-admin revoke-request c75bcdd5-cd44-4048-9c6a-b42e18b8451f reason=\"Security incident\"`")
	}

	requestID := args[0]

	// Extract reason from remaining args (default if not provided)
	reason := "Revoked by administrator"
	if len(args) > 1 {
		// Parse reason parameter
		params := parseKeyValueArgs(args[1:])
		if r, ok := params["reason"]; ok && r != "" {
			reason = r
		} else {
			// If not in key=value format, join all remaining args as reason
			reason = strings.Join(args[1:], " ")
		}
	}

	// Revoke the request
	request, err := requestService.RevokeRequest(ctx, requestID, adminID, adminName, reason)
	if err != nil {
		logger.LogError(ctx, "revoke_request", err, map[string]interface{}{
			"admin_id":   adminID,
			"request_id": requestID,
		})
		return slackResponse(fmt.Sprintf("❌ *Failed to Revoke Request*\n\n%s", err.Error()))
	}

	return slackResponse(fmt.Sprintf("✅ *Request Revoked*\n\n"+
		"Request `%s` has been revoked.\n\n"+
		"• User: <@%s>\n"+
		"• Host: `%s:%d`\n"+
		"• Revoked by: %s\n"+
		"• Reason: %s\n\n"+
		"The SSM document has been deleted and the user has been notified.",
		requestID,
		request.UserID,
		request.Host,
		request.Port,
		adminName,
		reason,
	))
}

// handleAddApprovalGroup adds a new approval group
func handleAddApprovalGroup(ctx context.Context, adminID, adminName string, argsText string) (events.APIGatewayProxyResponse, error) {
	params := parseKeyValueArgs([]string{argsText})

	groupID := params["group_id"]
	groupName := params["name"]
	groupType := params["type"]

	if groupID == "" || groupName == "" || groupType == "" {
		return slackResponse("❌ *Missing Parameters*\n\nUsage: `/ssm-admin add-approval-group group_id=<slack_group_id> name=<name> type=<security|manager>`\n\nExample: `/ssm-admin add-approval-group group_id=S12345678 name=\"Security Team\" type=security`")
	}

	// Validate type
	if groupType != "security" && groupType != "manager" {
		return slackResponse("❌ *Invalid Type*\n\nType must be either `security` or `manager`")
	}

	// Get Slack group handle
	handle, err := slackClient.GetUserGroupHandle(ctx, groupID)
	if err != nil {
		logger.LogError(ctx, "get_user_group_handle", err, map[string]interface{}{
			"group_id": groupID,
		})
		return slackResponse(fmt.Sprintf("❌ *Failed to Get Slack Group*\n\nCould not find Slack user group with ID `%s`. Please verify the group ID is correct.", groupID))
	}

	// Create approval group
	group := &models.ApprovalGroup{
		GroupID:     groupID,
		GroupName:   groupName,
		GroupType:   models.ApprovalGroupType(groupType),
		SlackHandle: handle,
		Active:      true,
		AddedBy:     adminID,
		AddedAt:     time.Now(),
		UpdatedAt:   time.Now(),
	}

	err = approvalGroupService.AddGroup(ctx, group, adminID, adminName)
	if err != nil {
		logger.LogError(ctx, "add_approval_group", err, map[string]interface{}{
			"admin_id": adminID,
			"group_id": groupID,
		})
		return slackResponse(fmt.Sprintf("❌ *Failed to Add Approval Group*\n\n%s", err.Error()))
	}

	return slackResponse(fmt.Sprintf("✅ *Approval Group Added*\n\n"+
		"*Name:* %s\n"+
		"*Type:* %s\n"+
		"*Slack Handle:* %s\n"+
		"*Group ID:* `%s`",
		groupName,
		groupType,
		handle,
		groupID,
	))
}

// handleListApprovalGroups lists all approval groups
func handleListApprovalGroups(ctx context.Context) (events.APIGatewayProxyResponse, error) {
	groups, err := approvalGroupService.ListAllGroups(ctx)
	if err != nil {
		logger.LogError(ctx, "list_approval_groups", err, map[string]interface{}{})
		return slackResponse("❌ *Failed to List Approval Groups*\n\nPlease try again later.")
	}

	if len(groups) == 0 {
		return slackResponse("📋 *Approval Groups*\n\nNo approval groups configured.")
	}

	message := "📋 *Approval Groups*\n\n"

	// Security groups
	message += "*Security Groups:*\n"
	hasSecurityGroups := false
	for _, group := range groups {
		if group.IsSecurity() {
			hasSecurityGroups = true
			status := "Active"
			if !group.Active {
				status = "Inactive"
			}
			message += fmt.Sprintf("• %s (%s) - %s\n", group.GroupName, group.SlackHandle, status)
			message += fmt.Sprintf("  ID: `%s`\n", group.GroupID)
		}
	}
	if !hasSecurityGroups {
		message += "• None configured\n"
	}
	message += "\n"

	// Manager groups
	message += "*Manager Groups:*\n"
	hasManagerGroups := false
	for _, group := range groups {
		if group.IsManager() {
			hasManagerGroups = true
			status := "Active"
			if !group.Active {
				status = "Inactive"
			}
			message += fmt.Sprintf("• %s (%s) - %s\n", group.GroupName, group.SlackHandle, status)
			message += fmt.Sprintf("  ID: `%s`\n", group.GroupID)
		}
	}
	if !hasManagerGroups {
		message += "• None configured\n"
	}

	return slackResponse(message)
}

// handleUpdateApprovalGroup updates an approval group
func handleUpdateApprovalGroup(ctx context.Context, adminID, adminName string, argsText string) (events.APIGatewayProxyResponse, error) {
	params := parseKeyValueArgs([]string{argsText})

	groupID := params["group_id"]
	if groupID == "" {
		return slackResponse("❌ *Missing Parameter*\n\nUsage: `/ssm-admin update-approval-group group_id=<id> [name=<name>] [active=<true|false>]`\n\nExample: `/ssm-admin update-approval-group group_id=S12345678 name=\"New Name\" active=true`")
	}

	updates := make(map[string]interface{})

	if name, ok := params["name"]; ok && name != "" {
		updates["group_name"] = name
	}

	if activeStr, ok := params["active"]; ok && activeStr != "" {
		active := activeStr == "true"
		updates["active"] = active
	}

	if len(updates) == 0 {
		return slackResponse("❌ *No Updates Specified*\n\nPlease specify at least one field to update: `name` or `active`")
	}

	err := approvalGroupService.UpdateGroup(ctx, groupID, updates, adminID, adminName)
	if err != nil {
		logger.LogError(ctx, "update_approval_group", err, map[string]interface{}{
			"admin_id": adminID,
			"group_id": groupID,
		})
		return slackResponse(fmt.Sprintf("❌ *Failed to Update Approval Group*\n\n%s", err.Error()))
	}

	return slackResponse(fmt.Sprintf("✅ *Approval Group Updated*\n\nGroup `%s` has been updated successfully.", groupID))
}

// handleRemoveApprovalGroup removes an approval group
func handleRemoveApprovalGroup(ctx context.Context, adminID, adminName string, argsText string) (events.APIGatewayProxyResponse, error) {
	args := strings.Fields(argsText)
	if len(args) < 1 {
		return slackResponse("❌ *Missing Parameter*\n\nUsage: `/ssm-admin remove-approval-group <group_id>`\n\nExample: `/ssm-admin remove-approval-group S12345678`")
	}

	groupID := args[0]

	// Confirm group exists
	group, err := approvalGroupService.GetGroup(ctx, groupID)
	if err != nil {
		logger.LogError(ctx, "get_approval_group", err, map[string]interface{}{
			"group_id": groupID,
		})
		return slackResponse(fmt.Sprintf("❌ *Group Not Found*\n\nCould not find approval group with ID `%s`", groupID))
	}

	err = approvalGroupService.RemoveGroup(ctx, groupID, adminID, adminName)
	if err != nil {
		logger.LogError(ctx, "remove_approval_group", err, map[string]interface{}{
			"admin_id": adminID,
			"group_id": groupID,
		})
		return slackResponse(fmt.Sprintf("❌ *Failed to Remove Approval Group*\n\n%s", err.Error()))
	}

	return slackResponse(fmt.Sprintf("✅ *Approval Group Removed*\n\n%s (%s) has been removed.", group.GroupName, group.SlackHandle))
}

// handleAuditLogs generates a CloudWatch Logs Insights query for audit logs
func handleAuditLogs(ctx context.Context, argsText string) (events.APIGatewayProxyResponse, error) {
	params := parseKeyValueArgs([]string{argsText})

	// Build CloudWatch Logs Insights query
	query := "fields @timestamp, event_type, actor.user_id, actor.user_name, target.request_id, details\n| sort @timestamp desc"

	// Add filters based on parameters
	filters := []string{}

	if requestID, ok := params["request_id"]; ok && requestID != "" {
		filters = append(filters, fmt.Sprintf("target.request_id = \"%s\"", requestID))
	}

	if userID, ok := params["user_id"]; ok && userID != "" {
		filters = append(filters, fmt.Sprintf("actor.user_id = \"%s\"", userID))
	}

	if eventType, ok := params["event_type"]; ok && eventType != "" {
		filters = append(filters, fmt.Sprintf("event_type = \"%s\"", eventType))
	}

	if len(filters) > 0 {
		query = "fields @timestamp, event_type, actor.user_id, actor.user_name, target.request_id, details\n| filter " + strings.Join(filters, " and ") + "\n| sort @timestamp desc"
	}

	// Get log group name from environment
	logGroup := os.Getenv("AUDIT_LOG_GROUP")
	if logGroup == "" {
		logGroup = "/aws/ssm-access-manager/audit"
	}

	message := "📊 *Audit Logs Query*\n\n"
	message += "*CloudWatch Logs Insights Query:*\n"
	message += "```\n" + query + "\n```\n\n"
	message += fmt.Sprintf("*Log Group:* `%s`\n\n", logGroup)
	message += "*How to Run:*\n"
	message += "1. Go to AWS CloudWatch Console\n"
	message += "2. Navigate to Logs > Insights\n"
	message += fmt.Sprintf("3. Select log group: `%s`\n", logGroup)
	message += "4. Paste the query above\n"
	message += "5. Select time range and click 'Run query'\n\n"
	message += "*Available Event Types:*\n"
	message += "• `request_created` - Access request created\n"
	message += "• `request_approved_security` - Security approval granted\n"
	message += "• `request_approved_manager` - Manager approval granted\n"
	message += "• `request_denied` - Request denied\n"
	message += "• `request_revoked` - Request revoked\n"
	message += "• `unauthorized_approval_attempt` - Unauthorized approval attempt\n"
	message += "• `approval_group_added` - Approval group added\n"
	message += "• `approval_group_updated` - Approval group updated\n"
	message += "• `approval_group_removed` - Approval group removed\n\n"
	message += "*Filter Examples:*\n"
	message += "`/ssm-admin audit-logs request_id=c75bcdd5-cd44-4048-9c6a-b42e18b8451f`\n"
	message += "`/ssm-admin audit-logs user_id=U12345678`\n"
	message += "`/ssm-admin audit-logs event_type=request_approved_security`"

	return slackResponse(message)
}

func main() {
	lambda.Start(handler)
}
