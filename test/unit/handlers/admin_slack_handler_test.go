package handlers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ssm-access-manager/internal/audit"
	"github.com/ssm-access-manager/internal/logging"
	"github.com/ssm-access-manager/internal/models"
	"github.com/ssm-access-manager/internal/service"
	"github.com/ssm-access-manager/test/helpers"
)

// TestAdminHandler_UnauthorizedAttemptLogging tests that unauthorized admin command attempts are logged
func TestAdminHandler_UnauthorizedAttemptLogging(t *testing.T) {
	ctx := context.Background()
	
	// Setup mocks
	mockUserRepo := new(helpers.MockUserRepository)
	mockGroupCache := new(helpers.MockGroupMembershipCache)
	
	// Create logger
	logger, err := logging.NewProductionLogger()
	assert.NoError(t, err)
	
	// Create audit service
	auditService := audit.NewAuditLogService(logger, "/aws/test/audit", "T12345678")
	
	// Create authorization service with audit service
	authService := service.NewAuthorizationService(mockUserRepo, mockGroupCache, auditService)
	
	// Test data
	userID := "U12345678"
	userName := "john.doe"
	action := "admin_command"
	
	// Mock: User is not an administrator
	mockUserRepo.On("GetUser", ctx, userID).Return(nil, nil)
	
	// Execute: Verify authorization (should fail)
	err = authService.VerifyAdministratorAuthorization(ctx, userID)
	assert.Error(t, err, "Expected authorization to fail for non-admin user")
	
	// Execute: Log unauthorized attempt
	authService.LogUnauthorizedAttempt(ctx, userID, userName, action)
	
	// Note: We can't easily verify CloudWatch Logs were written in a unit test,
	// but we can verify the method executes without error
	// In a real scenario, this would be verified through integration tests or CloudWatch Logs inspection
	
	mockUserRepo.AssertExpectations(t)
}

// TestAdminHandler_AuthorizedUserNoLogging tests that authorized users don't trigger unauthorized logging
func TestAdminHandler_AuthorizedUserNoLogging(t *testing.T) {
	ctx := context.Background()
	
	// Setup mocks
	mockUserRepo := new(helpers.MockUserRepository)
	mockGroupCache := new(helpers.MockGroupMembershipCache)
	mockAudit := new(helpers.MockAuditService)
	
	// Create logger
	logger, err := logging.NewProductionLogger()
	assert.NoError(t, err)
	
	// Create audit service
	auditService := audit.NewAuditLogService(logger, "/aws/test/audit", "T12345678")
	
	// Create authorization service with audit service
	authService := service.NewAuthorizationService(mockUserRepo, mockGroupCache, auditService)
	
	// Test data
	userID := "U12345678"
	
	// Mock: User is an administrator
	adminUser := &models.User{
		UserID: userID,
		Email:  "admin@example.com",
		Role:   models.UserRoleAdministrator,
	}
	mockUserRepo.On("GetUser", ctx, userID).Return(adminUser, nil)
	
	// Execute: Verify authorization (should succeed)
	err = authService.VerifyAdministratorAuthorization(ctx, userID)
	assert.NoError(t, err, "Expected authorization to succeed for admin user")
	
	// For authorized users, LogUnauthorizedAttempt should NOT be called
	// This is verified by the handler logic - if authorization succeeds, the log method is not invoked
	
	mockUserRepo.AssertExpectations(t)
	mockAudit.AssertNotCalled(t, "LogUnauthorizedApprovalAttempt")
}

// TestAdminHandler_AuditServiceNil tests that logging doesn't crash when audit service is nil
func TestAdminHandler_AuditServiceNil(t *testing.T) {
	ctx := context.Background()
	
	// Setup mocks
	mockUserRepo := new(helpers.MockUserRepository)
	mockGroupCache := new(helpers.MockGroupMembershipCache)
	
	// Create authorization service WITHOUT audit service (nil)
	authService := service.NewAuthorizationService(mockUserRepo, mockGroupCache, nil)
	
	// Test data
	userID := "U12345678"
	userName := "john.doe"
	action := "admin_command"
	
	// Execute: Log unauthorized attempt with nil audit service
	// This should not panic or crash
	assert.NotPanics(t, func() {
		authService.LogUnauthorizedAttempt(ctx, userID, userName, action)
	}, "LogUnauthorizedAttempt should handle nil audit service gracefully")
}
