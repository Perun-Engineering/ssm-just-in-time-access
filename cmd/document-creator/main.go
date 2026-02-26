package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	
	"github.com/ssm-access-manager/internal/logging"
	"github.com/ssm-access-manager/internal/repository"
	"github.com/ssm-access-manager/internal/service"
	"github.com/ssm-access-manager/internal/slack"
	"github.com/ssm-access-manager/internal/validation"
	awshelper "github.com/ssm-access-manager/pkg/aws"
)

var (
	logger          *logging.Logger
	slackClient     *slack.Client
	slackNotifier   *slack.Notifier
	documentService *service.SSMDocumentService
	requestRepo     *repository.RequestRepository
	documentRepo    *repository.DocumentRepository
	accountRepo     *repository.AccountRepository
)

// DocumentCreationEvent represents the event payload for document creation
type DocumentCreationEvent struct {
	RequestID string `json:"request_id"`
}

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

	// Initialize AWS clients
	ctx := context.Background()
	roleAssumer, err := awshelper.NewRoleAssumer(ctx)
	if err != nil {
		logger.Fatal("failed to create role assumer")
	}

	dynamoClient := dynamodb.NewFromConfig(roleAssumer.BaseConfig)

	// Initialize repositories
	requestsTable := os.Getenv("REQUESTS_TABLE")
	documentsTable := os.Getenv("DOCUMENTS_TABLE")
	accountsTable := os.Getenv("ACCOUNTS_TABLE")
	
	requestRepo = repository.NewRequestRepository(dynamoClient, requestsTable)
	documentRepo = repository.NewDocumentRepository(dynamoClient, documentsTable)
	accountRepo = repository.NewAccountRepository(dynamoClient, accountsTable)

	// Initialize services
	documentPrefix := os.Getenv("DOCUMENT_PREFIX")
	if documentPrefix == "" {
		documentPrefix = "PF" // Default: PortForwarding
	}
	nameGenerator := validation.NewDocumentNameGeneratorWithPrefix(documentPrefix)
	documentService = service.NewSSMDocumentService(documentRepo, roleAssumer, nameGenerator)
}

func handler(ctx context.Context, event events.CloudWatchEvent) error {
	// Parse the event detail
	var creationEvent DocumentCreationEvent
	err := json.Unmarshal(event.Detail, &creationEvent)
	if err != nil {
		logger.LogError(ctx, "parse_event", err, map[string]interface{}{
			"event": string(event.Detail),
		})
		return fmt.Errorf("failed to parse event: %w", err)
	}

	requestID := creationEvent.RequestID
	if requestID == "" {
		logger.Error("missing request ID in event")
		return fmt.Errorf("missing request ID")
	}

	// Retrieve the approved request
	request, err := requestRepo.GetRequestByID(ctx, requestID)
	if err != nil {
		logger.LogError(ctx, "get_request", err, map[string]interface{}{
			"request_id": requestID,
		})
		return fmt.Errorf("failed to get request: %w", err)
	}

	// Verify request is approved
	if !request.IsApproved() {
		logger.Warn("request is not approved")
		return fmt.Errorf("request %s is not approved (status: %s)", requestID, request.Status)
	}

	// Get target account configuration
	account, err := accountRepo.GetAccountByID(ctx, request.AccountID)
	if err != nil {
		logger.LogError(ctx, "get_account", err, map[string]interface{}{
			"account_id": request.AccountID,
		})
		
		errMsg := fmt.Sprintf("Failed to get account configuration: %s", err.Error())
		_ = slackNotifier.SendDocumentCreationFailure(ctx, request.UserID, request, errMsg)
		
		return fmt.Errorf("failed to get account: %w", err)
	}

	// Check if account exists
	if account == nil {
		logger.Warn(fmt.Sprintf("account %s not found", request.AccountID))
		
		errMsg := fmt.Sprintf("Account %s is not configured. Please contact an administrator to add this account.", request.AccountID)
		_ = slackNotifier.SendDocumentCreationFailure(ctx, request.UserID, request, errMsg)
		
		return fmt.Errorf("account %s not found", request.AccountID)
	}

	// Verify account is active
	if !account.IsActive() {
		logger.Warn("account is not active")
		
		errMsg := fmt.Sprintf("Account %s is not active", request.AccountID)
		_ = slackNotifier.SendDocumentCreationFailure(ctx, request.UserID, request, errMsg)
		
		return fmt.Errorf("account %s is not active", request.AccountID)
	}

	// Use first region as default if multiple regions are configured
	region := account.Regions[0]

	// Create SSM document
	document, err := documentService.CreateDocument(
		ctx,
		request,
		account.AccountID,
		region,
	)
	if err != nil {
		logger.LogError(ctx, "create_document", err, map[string]interface{}{
			"request_id": requestID,
			"account_id": account.AccountID,
			"host":       request.Host,
			"port":       request.Port,
		})
		
		errMsg := fmt.Sprintf("Failed to create SSM document: %s", err.Error())
		_ = slackNotifier.SendDocumentCreationFailure(ctx, request.UserID, request, errMsg)
		
		return fmt.Errorf("failed to create document: %w", err)
	}

	// Log document creation
	logger.LogDocumentCreation(
		ctx,
		document.DocumentID,
		document.DocumentName,
		document.AccountID,
		request.Username,
		document.Host,
		document.Port,
	)

	// Send success notification to user with account information
	err = slackNotifier.SendDocumentCreationSuccess(ctx, request.UserID, document, account)
	if err != nil {
		logger.Error("failed to send document creation success notification")
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
