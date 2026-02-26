package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go-v2/aws"
	
	"github.com/ssm-access-manager/internal/logging"
	"github.com/ssm-access-manager/internal/models"
	"github.com/ssm-access-manager/internal/repository"
	"github.com/ssm-access-manager/internal/service"
	"github.com/ssm-access-manager/internal/slack"
	"github.com/ssm-access-manager/internal/validation"
	awshelper "github.com/ssm-access-manager/pkg/aws"
)

var (
	logger              *logging.Logger
	slackClient         *slack.Client
	slackNotifier       *slack.Notifier
	interactionHandler  *slack.InteractionHandler
	requestService      *service.AccessRequestService
	authService         *service.AuthorizationService
	requestRepo         *repository.RequestRepository
	userRepo            *repository.UserRepository
	eventBridgeClient   *eventbridge.Client
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
	interactionHandler = slack.NewInteractionHandler(slackClient)

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
	
	requestRepo = repository.NewRequestRepository(dynamoClient, requestsTable)
	userRepo = repository.NewUserRepository(dynamoClient, usersTable)

	// Initialize services
	validator := validation.NewRequestValidator(90)
	authService = service.NewAuthorizationService(userRepo, nil, nil) // No group cache or audit service needed for approval handler
	requestService = service.NewAccessRequestService(requestRepo, validator, authService, nil) // No audit service needed
	
	// Configure self-approval setting (for testing purposes only)
	// WARNING: Should only be enabled in test/development environments
	allowSelfApproval := os.Getenv("ALLOW_SELF_APPROVAL")
	if allowSelfApproval == "true" {
		logger.Warn("ALLOW_SELF_APPROVAL is enabled - this should only be used for testing!")
		requestService.SetAllowSelfApproval(true)
	}
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Verify Slack signature
	headers := make(http.Header)
	for k, v := range request.Headers {
		headers.Set(k, v)
	}
	
	err := slackClient.VerifySignature(headers, request.Body)
	if err != nil {
		logger.Warn("invalid Slack signature")
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       "Unauthorized",
		}, nil
	}

	// Parse form-encoded body
	values, err := url.ParseQuery(request.Body)
	if err != nil {
		logger.Error("failed to parse request body")
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Bad Request",
		}, nil
	}

	payload := values.Get("payload")
	if payload == "" {
		logger.Error("missing payload")
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Bad Request",
		}, nil
	}

	// Parse interaction callback
	callback, err := slack.ParseInteractionCallback(payload)
	if err != nil {
		logger.Error("failed to parse interaction callback")
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Bad Request",
		}, nil
	}

	// Get request ID from action value
	requestID := callback.GetActionValue()
	if requestID == "" {
		logger.Error("missing request ID")
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Bad Request",
		}, nil
	}

	// Get approver info
	approverID := callback.User.ID
	approverName := callback.User.Name

	// Handle approval or denial
	var accessRequest *models.AccessRequest
	var decision string

	if callback.IsApproval() {
		accessRequest, err = requestService.ApproveRequest(ctx, requestID, approverID, approverName)
		decision = "approved"
	} else if callback.IsDenial() {
		reason := "Denied by manager"
		accessRequest, err = requestService.DenyRequest(ctx, requestID, approverID, approverName, reason)
		decision = "denied"
	} else {
		logger.Error("unknown action")
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Bad Request",
		}, nil
	}

	if err != nil {
		logger.LogError(ctx, "approval_decision", err, map[string]interface{}{
			"request_id":  requestID,
			"approver_id": approverID,
			"decision":    decision,
		})
		
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       fmt.Sprintf("Failed to process decision: %s", err.Error()),
		}, nil
	}

	// Log the decision
	logger.LogApprovalDecision(ctx, requestID, approverName, approverID, decision)

	// Update the original message
	err = interactionHandler.UpdateMessageWithResult(ctx, callback, callback.IsApproval(), approverName)
	if err != nil {
		logger.Error("failed to update message")
	}

	// Send notification to user
	if callback.IsApproval() {
		err = slackNotifier.SendApprovalConfirmation(ctx, accessRequest.UserID, accessRequest)
		
		// Publish EventBridge event to trigger document creation
		err = publishApprovalEvent(ctx, accessRequest.RequestID)
		if err != nil {
			logger.LogError(ctx, "publish_approval_event", err, map[string]interface{}{
				"request_id": requestID,
			})
			// Don't fail the request, just log the error
		}
	} else {
		reason := "Denied by manager"
		if accessRequest.DenialReason != nil {
			reason = *accessRequest.DenialReason
		}
		err = slackNotifier.SendDenialNotification(ctx, accessRequest.UserID, accessRequest, reason)
	}

	if err != nil {
		logger.Error("failed to send notification to user")
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "Decision processed successfully",
	}, nil
}

// publishApprovalEvent publishes an event to EventBridge to trigger document creation
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

func main() {
	lambda.Start(handler)
}
