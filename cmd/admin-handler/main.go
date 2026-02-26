package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	
	"github.com/ssm-access-manager/internal/logging"
	"github.com/ssm-access-manager/internal/repository"
	"github.com/ssm-access-manager/internal/service"
	"github.com/ssm-access-manager/internal/validation"
	awshelper "github.com/ssm-access-manager/pkg/aws"
)

var (
	logger         *logging.Logger
	authService    *service.AuthorizationService
	accountService *service.AccountService
	userRepo       *repository.UserRepository
)

func init() {
	var err error
	
	// Initialize logger
	logger, err = logging.NewProductionLogger()
	if err != nil {
		panic(fmt.Sprintf("failed to create logger: %v", err))
	}

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
	
	userRepo = repository.NewUserRepository(dynamoClient, usersTable)
	accountRepo := repository.NewAccountRepository(dynamoClient, accountsTable)

	// Initialize services
	validator := validation.NewRequestValidator(90)
	authService = service.NewAuthorizationService(userRepo, nil, nil) // No group cache or audit service needed
	accountService = service.NewAccountService(accountRepo, validator, roleAssumer, authService)
}

// AdminRequest represents an admin operation request
type AdminRequest struct {
	Action      string   `json:"action"`
	AdminID     string   `json:"admin_id"`
	UserID      string   `json:"user_id,omitempty"`
	Email       string   `json:"email,omitempty"`
	AccountID   string   `json:"account_id,omitempty"`
	AccountName string   `json:"account_name,omitempty"`
	RoleName    string   `json:"role_name,omitempty"`
	Regions     []string `json:"regions,omitempty"`
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Parse request body
	var adminReq AdminRequest
	err := json.Unmarshal([]byte(request.Body), &adminReq)
	if err != nil {
		return errorResponse(http.StatusBadRequest, "Invalid request body")
	}

	// Verify administrator authorization
	err = authService.VerifyAdministratorAuthorization(ctx, adminReq.AdminID)
	if err != nil {
		authService.LogUnauthorizedAttempt(ctx, adminReq.AdminID, adminReq.AdminID, fmt.Sprintf("admin_action_%s", adminReq.Action))
		return errorResponse(http.StatusForbidden, "Unauthorized: administrator access required")
	}

	// Route to appropriate handler
	switch adminReq.Action {
	case "add_administrator":
		return handleAddAdministrator(ctx, adminReq)
	case "remove_administrator":
		return handleRemoveAdministrator(ctx, adminReq)
	case "add_account":
		return handleAddAccount(ctx, adminReq)
	case "remove_account":
		return handleRemoveAccount(ctx, adminReq)
	case "list_administrators":
		return handleListAdministrators(ctx)
	case "list_accounts":
		return handleListAccounts(ctx)
	default:
		return errorResponse(http.StatusBadRequest, fmt.Sprintf("Unknown action: %s (manager actions have been deprecated - use approval groups instead)", adminReq.Action))
	}
}

func handleAddAdministrator(ctx context.Context, req AdminRequest) (events.APIGatewayProxyResponse, error) {
	if req.UserID == "" || req.Email == "" {
		return errorResponse(http.StatusBadRequest, "user_id and email are required")
	}

	// Use UserID as username if not provided
	username := req.UserID

	err := authService.AddAdministrator(ctx, req.UserID, username, req.Email, req.AdminID)
	if err != nil {
		logger.LogError(ctx, "add_administrator", err, map[string]interface{}{
			"admin_id": req.AdminID,
			"user_id":  req.UserID,
		})
		return errorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to add administrator: %s", err.Error()))
	}

	return successResponse(map[string]interface{}{
		"message": "Administrator added successfully",
		"user_id": req.UserID,
	})
}

func handleRemoveAdministrator(ctx context.Context, req AdminRequest) (events.APIGatewayProxyResponse, error) {
	if req.UserID == "" {
		return errorResponse(http.StatusBadRequest, "user_id is required")
	}

	err := authService.RemoveAdministrator(ctx, req.UserID, req.AdminID)
	if err != nil {
		logger.LogError(ctx, "remove_administrator", err, map[string]interface{}{
			"admin_id": req.AdminID,
			"user_id":  req.UserID,
		})
		return errorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to remove administrator: %s", err.Error()))
	}

	return successResponse(map[string]interface{}{
		"message": "Administrator removed successfully",
		"user_id": req.UserID,
	})
}

func handleAddAccount(ctx context.Context, req AdminRequest) (events.APIGatewayProxyResponse, error) {
	if req.AccountID == "" || req.RoleName == "" || len(req.Regions) == 0 {
		return errorResponse(http.StatusBadRequest, "account_id, role_name, and regions are required")
	}

	// Use AccountID as AccountName if not provided
	accountName := req.AccountName
	if accountName == "" {
		accountName = req.AccountID
	}

	account, err := accountService.AddAccount(ctx, req.AccountID, accountName, req.RoleName, req.Regions, "", req.AdminID)
	if err != nil {
		logger.LogError(ctx, "add_account", err, map[string]interface{}{
			"admin_id":   req.AdminID,
			"account_id": req.AccountID,
		})
		return errorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to add account: %s", err.Error()))
	}

	return successResponse(map[string]interface{}{
		"message":    "Account added successfully",
		"account_id": account.AccountID,
	})
}

func handleRemoveAccount(ctx context.Context, req AdminRequest) (events.APIGatewayProxyResponse, error) {
	if req.AccountID == "" {
		return errorResponse(http.StatusBadRequest, "account_id is required")
	}

	err := accountService.RemoveAccount(ctx, req.AccountID, req.AdminID)
	if err != nil {
		logger.LogError(ctx, "remove_account", err, map[string]interface{}{
			"admin_id":   req.AdminID,
			"account_id": req.AccountID,
		})
		return errorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to remove account: %s", err.Error()))
	}

	return successResponse(map[string]interface{}{
		"message":    "Account removed successfully",
		"account_id": req.AccountID,
	})
}

func handleListAdministrators(ctx context.Context) (events.APIGatewayProxyResponse, error) {
	admins, err := authService.GetAllAdministrators(ctx)
	if err != nil {
		logger.LogError(ctx, "list_administrators", err, map[string]interface{}{})
		return errorResponse(http.StatusInternalServerError, "Failed to list administrators")
	}

	return successResponse(map[string]interface{}{
		"administrators": admins,
		"count":          len(admins),
	})
}

func handleListAccounts(ctx context.Context) (events.APIGatewayProxyResponse, error) {
	accounts, err := accountService.ListAccounts(ctx)
	if err != nil {
		logger.LogError(ctx, "list_accounts", err, map[string]interface{}{})
		return errorResponse(http.StatusInternalServerError, "Failed to list accounts")
	}

	return successResponse(map[string]interface{}{
		"accounts": accounts,
		"count":    len(accounts),
	})
}

func successResponse(data interface{}) (events.APIGatewayProxyResponse, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "Failed to marshal response")
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(body),
	}, nil
}

func errorResponse(statusCode int, message string) (events.APIGatewayProxyResponse, error) {
	body, _ := json.Marshal(map[string]string{
		"error": message,
	})

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(body),
	}, nil
}

func main() {
	lambda.Start(handler)
}
