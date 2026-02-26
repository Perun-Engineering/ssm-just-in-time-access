package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	
	"github.com/ssm-access-manager/internal/logging"
	"github.com/ssm-access-manager/internal/models"
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
	documentRepo    *repository.DocumentRepository
	userRepo        *repository.UserRepository
	authService     *service.AuthorizationService
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

	// Initialize AWS clients
	ctx := context.Background()
	roleAssumer, err := awshelper.NewRoleAssumer(ctx)
	if err != nil {
		logger.Fatal("failed to create role assumer")
	}

	dynamoClient := dynamodb.NewFromConfig(roleAssumer.BaseConfig)

	// Initialize repositories
	documentsTable := os.Getenv("DOCUMENTS_TABLE")
	usersTable := os.Getenv("USERS_TABLE")
	
	documentRepo = repository.NewDocumentRepository(dynamoClient, documentsTable)
	userRepo = repository.NewUserRepository(dynamoClient, usersTable)

	// Initialize services
	documentPrefix := os.Getenv("DOCUMENT_PREFIX")
	if documentPrefix == "" {
		documentPrefix = "PF" // Default: PortForwarding
	}
	nameGenerator := validation.NewDocumentNameGeneratorWithPrefix(documentPrefix)
	documentService = service.NewSSMDocumentService(documentRepo, roleAssumer, nameGenerator)
	authService = service.NewAuthorizationService(userRepo, nil, nil) // No group cache or audit service needed
}

func handler(ctx context.Context) error {
	logger.Info("starting expiration cleanup")

	// Get all expired documents
	expiredDocs, err := documentRepo.GetExpiredDocuments(ctx)
	if err != nil {
		logger.LogError(ctx, "get_expired_documents", err, map[string]interface{}{})
		return fmt.Errorf("failed to get expired documents: %w", err)
	}

	if len(expiredDocs) == 0 {
		logger.Info("no expired documents found")
		return nil
	}

	logger.Info(fmt.Sprintf("found %d expired documents", len(expiredDocs)))

	// Track results
	var successCount, failureCount int
	var persistentFailures []string

	// Process each expired document
	for _, doc := range expiredDocs {
		err := processExpiredDocument(ctx, doc)
		if err != nil {
			failureCount++
			persistentFailures = append(persistentFailures, fmt.Sprintf("%s: %s", doc.DocumentName, err.Error()))
			logger.LogError(ctx, "delete_expired_document", err, map[string]interface{}{
				"document_id":   doc.DocumentID,
				"document_name": doc.DocumentName,
				"account_id":    doc.AccountID,
			})
		} else {
			successCount++
		}
	}

	// Generate expiration report
	logger.Info(fmt.Sprintf("expiration cleanup complete: %d succeeded, %d failed", successCount, failureCount))

	// Alert administrators on persistent failures
	if len(persistentFailures) > 0 {
		err := alertAdministrators(ctx, persistentFailures)
		if err != nil {
			logger.Error("failed to alert administrators")
		}
	}

	return nil
}

func processExpiredDocument(ctx context.Context, doc *models.SSMDocument) error {
	// Delete the SSM document
	err := documentService.DeleteDocument(ctx, doc)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	// Log document deletion
	logger.LogDocumentDeletion(ctx, doc.DocumentID, doc.DocumentName, doc.AccountID, "expired")

	// Send expiration notification to user
	// Note: We need to get the user's Slack ID from the username
	// For now, we'll use the username as the user ID (this should be improved)
	err = slackNotifier.SendExpirationNotification(ctx, doc.Username, doc)
	if err != nil {
		logger.Error("failed to send expiration notification")
		// Don't fail the entire operation if notification fails
	}

	return nil
}

func alertAdministrators(ctx context.Context, failures []string) error {
	// Get all administrators
	admins, err := authService.GetAllAdministrators(ctx)
	if err != nil {
		return fmt.Errorf("failed to get administrators: %w", err)
	}

	if len(admins) == 0 {
		logger.Warn("no administrators found to alert")
		return nil
	}

	// Build alert message
	alertMsg := fmt.Sprintf("Expiration cleanup encountered %d persistent failures:\n\n", len(failures))
	for _, failure := range failures {
		alertMsg += fmt.Sprintf("• %s\n", failure)
	}
	alertMsg += fmt.Sprintf("\nTime: %s", time.Now().Format("2006-01-02 15:04:05 MST"))

	// Get admin user IDs
	var adminUserIDs []string
	for _, admin := range admins {
		adminUserIDs = append(adminUserIDs, admin.UserID)
	}

	// Send alert
	err = slackNotifier.SendAdminAlert(ctx, adminUserIDs, alertMsg)
	if err != nil {
		return fmt.Errorf("failed to send admin alert: %w", err)
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
