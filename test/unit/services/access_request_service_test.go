package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/ssm-access-manager/internal/models"
	"github.com/ssm-access-manager/internal/service"
	"github.com/ssm-access-manager/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestAccessRequestService_CreateRequest_WithManagerGroup tests creating a request with manager group
func TestAccessRequestService_CreateRequest_WithManagerGroup(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(helpers.MockRequestRepository)
	mockValidator := new(helpers.MockRequestValidator)
	mockAuth := new(helpers.MockAuthorizationService)
	mockAudit := new(helpers.MockAuditService)

	svc := service.NewAccessRequestService(mockRepo, mockValidator, mockAuth, mockAudit)

	// Setup test data
	username := "test.user"
	userID := "U123456"
	host := "prod-db-01"
	port := 5432
	accountID := "123456789012"
	expirationDate := time.Now().Add(24 * time.Hour)
	managerGroupID := helpers.ManagerGroupID
	managerGroupName := "SRE Cloud OPS"

	// Mock validations
	mockValidator.On("ValidateHost", host).Return(&models.ValidationResult{IsValid: true})
	mockValidator.On("ValidatePort", port).Return(&models.ValidationResult{IsValid: true})
	mockValidator.On("ValidateExpirationDate", mock.Anything).Return(&models.ValidationResult{IsValid: true})
	mockValidator.On("ValidateUsername", username).Return(&models.ValidationResult{IsValid: true})
	mockValidator.On("ValidateAccountID", accountID).Return(&models.ValidationResult{IsValid: true})

	// Mock save operation
	mockRepo.On("SaveRequest", ctx, mock.MatchedBy(func(r *models.AccessRequest) bool {
		return r.Username == username &&
			r.UserID == userID &&
			r.Host == host &&
			r.Port == port &&
			r.AccountID == accountID &&
			r.ManagerGroupID == managerGroupID &&
			r.ManagerGroupName == managerGroupName &&
			r.Reason == "Test reason" &&
			r.Status == models.RequestStatusPending
	})).Return(nil)

	// Mock audit logging
	mockAudit.On("LogRequestCreated", ctx, mock.Anything).Return()

	// Execute
	request, err := svc.CreateRequest(ctx, username, userID, host, port, accountID, expirationDate, managerGroupID, managerGroupName, "Test reason")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, request)
	assert.NotEmpty(t, request.RequestID)
	assert.Equal(t, username, request.Username)
	assert.Equal(t, userID, request.UserID)
	assert.Equal(t, host, request.Host)
	assert.Equal(t, port, request.Port)
	assert.Equal(t, accountID, request.AccountID)
	assert.Equal(t, managerGroupID, request.ManagerGroupID)
	assert.Equal(t, managerGroupName, request.ManagerGroupName)
	assert.Equal(t, "Test reason", request.Reason)
	assert.Equal(t, models.RequestStatusPending, request.Status)
	assert.NotZero(t, request.CreatedAt)
	assert.NotZero(t, request.UpdatedAt)

	mockValidator.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockAudit.AssertExpectations(t)
}

// TestAccessRequestService_CreateRequest_ValidatesRequiredFields tests validation enforcement
func TestAccessRequestService_CreateRequest_ValidatesRequiredFields(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		host          string
		port          int
		setupMocks    func(*helpers.MockRequestValidator)
		expectedError string
	}{
		{
			name: "invalid host",
			host: "invalid",
			port: 5432,
			setupMocks: func(validator *helpers.MockRequestValidator) {
				validator.On("ValidateHost", "invalid").Return(&models.ValidationResult{
					IsValid:      false,
					ErrorMessage: "invalid hostname",
				})
			},
			expectedError: "invalid host",
		},
		{
			name: "invalid port",
			host: "valid-host",
			port: 99999,
			setupMocks: func(validator *helpers.MockRequestValidator) {
				validator.On("ValidateHost", "valid-host").Return(&models.ValidationResult{IsValid: true})
				validator.On("ValidatePort", 99999).Return(&models.ValidationResult{
					IsValid:      false,
					ErrorMessage: "port out of range",
				})
			},
			expectedError: "invalid port",
		},
		{
			name: "invalid expiration date",
			host: "valid-host",
			port: 5432,
			setupMocks: func(validator *helpers.MockRequestValidator) {
				validator.On("ValidateHost", "valid-host").Return(&models.ValidationResult{IsValid: true})
				validator.On("ValidatePort", 5432).Return(&models.ValidationResult{IsValid: true})
				validator.On("ValidateExpirationDate", mock.Anything).Return(&models.ValidationResult{
					IsValid:      false,
					ErrorMessage: "expiration date too far in future",
				})
			},
			expectedError: "invalid expiration date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh mocks for each subtest
			mockRepo := new(helpers.MockRequestRepository)
			mockValidator := new(helpers.MockRequestValidator)
			mockAuth := new(helpers.MockAuthorizationService)
			mockAudit := new(helpers.MockAuditService)

			svc := service.NewAccessRequestService(mockRepo, mockValidator, mockAuth, mockAudit)

			tt.setupMocks(mockValidator)

			// Execute
			request, err := svc.CreateRequest(
				ctx,
				"test.user",
				"U123456",
				tt.host,
				tt.port,
				"123456789012",
				time.Now().Add(24*time.Hour),
				helpers.ManagerGroupID,
				"Manager Group",
				"Test reason",
			)

			// Assert
			assert.Error(t, err)
			assert.Nil(t, request)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

// TestAccessRequestService_ApproveRequestSecurity_UpdatesStatus tests security approval
func TestAccessRequestService_ApproveRequestSecurity_UpdatesStatus(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(helpers.MockRequestRepository)
	mockValidator := new(helpers.MockRequestValidator)
	mockAuth := new(helpers.MockAuthorizationService)
	mockAudit := new(helpers.MockAuditService)

	svc := service.NewAccessRequestService(mockRepo, mockValidator, mockAuth, mockAudit)

	// Create a pending request
	request := helpers.CreateTestRequest(helpers.ManagerGroupID, "Manager Group")
	approverID := "U789012"
	approverName := "security.approver"

	// Mock get request
	mockRepo.On("GetRequestByID", ctx, request.RequestID).Return(request, nil)

	// Mock save request
	mockRepo.On("SaveRequest", ctx, mock.MatchedBy(func(r *models.AccessRequest) bool {
		return r.RequestID == request.RequestID &&
			r.Status == models.RequestStatusPartiallyApproved &&
			r.SecurityApproverID != nil &&
			*r.SecurityApproverID == approverID
	})).Return(nil)

	// Mock audit logging
	mockAudit.On("LogSecurityApproval", ctx, approverID, approverName, mock.Anything).Return()

	// Execute
	updatedRequest, err := svc.ApproveRequestSecurity(ctx, request.RequestID, approverID, approverName)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, updatedRequest)
	assert.Equal(t, models.RequestStatusPartiallyApproved, updatedRequest.Status)
	assert.True(t, updatedRequest.HasSecurityApproval())
	assert.False(t, updatedRequest.HasManagerApproval())
	assert.Equal(t, approverID, *updatedRequest.SecurityApproverID)
	assert.Equal(t, approverName, *updatedRequest.SecurityApproverName)

	mockRepo.AssertExpectations(t)
	mockAudit.AssertExpectations(t)
}

// TestAccessRequestService_ApproveRequestSecurity_PreventsDuplicateApproval tests duplicate prevention
func TestAccessRequestService_ApproveRequestSecurity_PreventsDuplicateApproval(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(helpers.MockRequestRepository)
	mockValidator := new(helpers.MockRequestValidator)
	mockAuth := new(helpers.MockAuthorizationService)
	mockAudit := new(helpers.MockAuditService)

	svc := service.NewAccessRequestService(mockRepo, mockValidator, mockAuth, mockAudit)

	// Create a request with existing security approval
	request := helpers.CreateTestRequest(helpers.ManagerGroupID, "Manager Group")
	existingApproverID := "U111111"
	request.SecurityApproverID = &existingApproverID

	// Mock get request
	mockRepo.On("GetRequestByID", ctx, request.RequestID).Return(request, nil)

	// Execute
	updatedRequest, err := svc.ApproveRequestSecurity(ctx, request.RequestID, "U222222", "another.approver")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, updatedRequest)
	assert.Contains(t, err.Error(), "security approval already granted")

	mockRepo.AssertNotCalled(t, "SaveRequest")
	mockAudit.AssertNotCalled(t, "LogSecurityApproval")
}

// TestAccessRequestService_ApproveRequestSecurity_PreventsSelfApproval tests self-approval prevention
func TestAccessRequestService_ApproveRequestSecurity_PreventsSelfApproval(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(helpers.MockRequestRepository)
	mockValidator := new(helpers.MockRequestValidator)
	mockAuth := new(helpers.MockAuthorizationService)
	mockAudit := new(helpers.MockAuditService)

	svc := service.NewAccessRequestService(mockRepo, mockValidator, mockAuth, mockAudit)

	// Create a request where the requester will try to approve their own request
	request := helpers.CreateTestRequest(helpers.ManagerGroupID, "Manager Group")
	requesterID := request.UserID
	requesterName := "self.approver"

	// Mock get request
	mockRepo.On("GetRequestByID", ctx, request.RequestID).Return(request, nil)

	// Mock audit logging for self-approval attempt
	mockAudit.On("LogSelfApprovalAttempt", ctx, requesterID, requesterName, request.RequestID).Return()

	// Execute - try to approve with same user ID as requester
	updatedRequest, err := svc.ApproveRequestSecurity(ctx, request.RequestID, requesterID, requesterName)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, updatedRequest)
	assert.Contains(t, err.Error(), "you cannot approve your own access request")

	// Verify audit log was called
	mockAudit.AssertExpectations(t)
	mockAudit.AssertCalled(t, "LogSelfApprovalAttempt", ctx, requesterID, requesterName, request.RequestID)

	// Verify request was not saved
	mockRepo.AssertNotCalled(t, "SaveRequest")
	mockAudit.AssertNotCalled(t, "LogSecurityApproval")
}

// TestAccessRequestService_ApproveRequestManager_UpdatesStatus tests manager approval
func TestAccessRequestService_ApproveRequestManager_UpdatesStatus(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(helpers.MockRequestRepository)
	mockValidator := new(helpers.MockRequestValidator)
	mockAuth := new(helpers.MockAuthorizationService)
	mockAudit := new(helpers.MockAuditService)

	svc := service.NewAccessRequestService(mockRepo, mockValidator, mockAuth, mockAudit)

	// Create a pending request
	request := helpers.CreateTestRequest(helpers.ManagerGroupID, "Manager Group")
	approverID := "U789012"
	approverName := "manager.approver"

	// Mock get request
	mockRepo.On("GetRequestByID", ctx, request.RequestID).Return(request, nil)

	// Mock save request
	mockRepo.On("SaveRequest", ctx, mock.MatchedBy(func(r *models.AccessRequest) bool {
		return r.RequestID == request.RequestID &&
			r.Status == models.RequestStatusPartiallyApproved &&
			r.ManagerApproverID != nil &&
			*r.ManagerApproverID == approverID
	})).Return(nil)

	// Mock audit logging
	mockAudit.On("LogManagerApproval", ctx, approverID, approverName, mock.Anything).Return()

	// Execute
	updatedRequest, err := svc.ApproveRequestManager(ctx, request.RequestID, approverID, approverName)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, updatedRequest)
	assert.Equal(t, models.RequestStatusPartiallyApproved, updatedRequest.Status)
	assert.False(t, updatedRequest.HasSecurityApproval())
	assert.True(t, updatedRequest.HasManagerApproval())
	assert.Equal(t, approverID, *updatedRequest.ManagerApproverID)
	assert.Equal(t, approverName, *updatedRequest.ManagerApproverName)

	mockRepo.AssertExpectations(t)
	mockAudit.AssertExpectations(t)
}

// TestAccessRequestService_ApproveRequestManager_PreventsSelfApproval tests self-approval prevention for manager approval
func TestAccessRequestService_ApproveRequestManager_PreventsSelfApproval(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(helpers.MockRequestRepository)
	mockValidator := new(helpers.MockRequestValidator)
	mockAuth := new(helpers.MockAuthorizationService)
	mockAudit := new(helpers.MockAuditService)

	svc := service.NewAccessRequestService(mockRepo, mockValidator, mockAuth, mockAudit)

	// Create a request where the requester will try to approve their own request
	request := helpers.CreateTestRequest(helpers.ManagerGroupID, "Manager Group")
	requesterID := request.UserID
	requesterName := "self.approver"

	// Mock get request
	mockRepo.On("GetRequestByID", ctx, request.RequestID).Return(request, nil)

	// Mock audit logging for self-approval attempt
	mockAudit.On("LogSelfApprovalAttempt", ctx, requesterID, requesterName, request.RequestID).Return()

	// Execute - try to approve with same user ID as requester
	updatedRequest, err := svc.ApproveRequestManager(ctx, request.RequestID, requesterID, requesterName)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, updatedRequest)
	assert.Contains(t, err.Error(), "you cannot approve your own access request")

	// Verify audit log was called
	mockAudit.AssertExpectations(t)
	mockAudit.AssertCalled(t, "LogSelfApprovalAttempt", ctx, requesterID, requesterName, request.RequestID)

	// Verify request was not saved
	mockRepo.AssertNotCalled(t, "SaveRequest")
	mockAudit.AssertNotCalled(t, "LogManagerApproval")
}

// TestAccessRequestService_StatusTransition_PendingToPartiallyApproved tests first approval
func TestAccessRequestService_StatusTransition_PendingToPartiallyApproved(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(helpers.MockRequestRepository)
	mockValidator := new(helpers.MockRequestValidator)
	mockAuth := new(helpers.MockAuthorizationService)
	mockAudit := new(helpers.MockAuditService)

	svc := service.NewAccessRequestService(mockRepo, mockValidator, mockAuth, mockAudit)

	// Create a pending request
	request := helpers.CreateTestRequest(helpers.ManagerGroupID, "Manager Group")
	assert.Equal(t, models.RequestStatusPending, request.Status)

	// Mock get request
	mockRepo.On("GetRequestByID", ctx, request.RequestID).Return(request, nil)
	mockRepo.On("SaveRequest", ctx, mock.Anything).Return(nil)
	mockAudit.On("LogSecurityApproval", ctx, mock.Anything, mock.Anything, mock.Anything).Return()

	// Execute - grant security approval
	updatedRequest, err := svc.ApproveRequestSecurity(ctx, request.RequestID, "U111111", "security.approver")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, models.RequestStatusPartiallyApproved, updatedRequest.Status)
}

// TestAccessRequestService_StatusTransition_PartiallyApprovedToApproved tests second approval
func TestAccessRequestService_StatusTransition_PartiallyApprovedToApproved(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(helpers.MockRequestRepository)
	mockValidator := new(helpers.MockRequestValidator)
	mockAuth := new(helpers.MockAuthorizationService)
	mockAudit := new(helpers.MockAuditService)

	svc := service.NewAccessRequestService(mockRepo, mockValidator, mockAuth, mockAudit)

	// Create a partially approved request (security approval already granted)
	request := helpers.CreateTestRequest(helpers.ManagerGroupID, "Manager Group")
	securityApproverID := "U111111"
	request.SecurityApproverID = &securityApproverID
	request.Status = models.RequestStatusPartiallyApproved

	// Mock get request
	mockRepo.On("GetRequestByID", ctx, request.RequestID).Return(request, nil)
	mockRepo.On("SaveRequest", ctx, mock.MatchedBy(func(r *models.AccessRequest) bool {
		return r.Status == models.RequestStatusApproved
	})).Return(nil)
	mockAudit.On("LogManagerApproval", ctx, mock.Anything, mock.Anything, mock.Anything).Return()

	// Execute - grant manager approval
	updatedRequest, err := svc.ApproveRequestManager(ctx, request.RequestID, "U222222", "manager.approver")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, models.RequestStatusApproved, updatedRequest.Status)
	assert.True(t, updatedRequest.HasSecurityApproval())
	assert.True(t, updatedRequest.HasManagerApproval())
	assert.True(t, updatedRequest.IsFullyApproved())
}

// TestAccessRequestService_AuditLogging tests audit logging for all operations
func TestAccessRequestService_AuditLogging(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		operation func(*service.AccessRequestService, *helpers.MockRequestRepository, *helpers.MockRequestValidator, *helpers.MockAuthorizationService, *helpers.MockAuditService) error
		auditCall string
	}{
		{
			name: "CreateRequest logs",
			operation: func(svc *service.AccessRequestService, repo *helpers.MockRequestRepository, validator *helpers.MockRequestValidator, auth *helpers.MockAuthorizationService, audit *helpers.MockAuditService) error {
				validator.On("ValidateHost", mock.Anything).Return(&models.ValidationResult{IsValid: true})
				validator.On("ValidatePort", mock.Anything).Return(&models.ValidationResult{IsValid: true})
				validator.On("ValidateExpirationDate", mock.Anything).Return(&models.ValidationResult{IsValid: true})
				validator.On("ValidateUsername", mock.Anything).Return(&models.ValidationResult{IsValid: true})
				validator.On("ValidateAccountID", mock.Anything).Return(&models.ValidationResult{IsValid: true})
				repo.On("SaveRequest", ctx, mock.Anything).Return(nil)
				audit.On("LogRequestCreated", ctx, mock.Anything).Return()
				_, err := svc.CreateRequest(ctx, "user", "U123", "host", 22, "123456789012", time.Now().Add(24*time.Hour), "S123", "Group", "Test reason")
				return err
			},
			auditCall: "LogRequestCreated",
		},
		{
			name: "ApproveRequestSecurity logs",
			operation: func(svc *service.AccessRequestService, repo *helpers.MockRequestRepository, validator *helpers.MockRequestValidator, auth *helpers.MockAuthorizationService, audit *helpers.MockAuditService) error {
				request := helpers.CreateTestRequest(helpers.ManagerGroupID, "Manager Group")
				repo.On("GetRequestByID", ctx, request.RequestID).Return(request, nil)
				repo.On("SaveRequest", ctx, mock.Anything).Return(nil)
				audit.On("LogSecurityApproval", ctx, mock.Anything, mock.Anything, mock.Anything).Return()
				_, err := svc.ApproveRequestSecurity(ctx, request.RequestID, "U123", "approver")
				return err
			},
			auditCall: "LogSecurityApproval",
		},
		{
			name: "ApproveRequestManager logs",
			operation: func(svc *service.AccessRequestService, repo *helpers.MockRequestRepository, validator *helpers.MockRequestValidator, auth *helpers.MockAuthorizationService, audit *helpers.MockAuditService) error {
				request := helpers.CreateTestRequest(helpers.ManagerGroupID, "Manager Group")
				repo.On("GetRequestByID", ctx, request.RequestID).Return(request, nil)
				repo.On("SaveRequest", ctx, mock.Anything).Return(nil)
				audit.On("LogManagerApproval", ctx, mock.Anything, mock.Anything, mock.Anything).Return()
				_, err := svc.ApproveRequestManager(ctx, request.RequestID, "U123", "approver")
				return err
			},
			auditCall: "LogManagerApproval",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(helpers.MockRequestRepository)
			mockValidator := new(helpers.MockRequestValidator)
			mockAuth := new(helpers.MockAuthorizationService)
			mockAudit := new(helpers.MockAuditService)
			svc := service.NewAccessRequestService(mockRepo, mockValidator, mockAuth, mockAudit)

			err := tt.operation(svc, mockRepo, mockValidator, mockAuth, mockAudit)

			assert.NoError(t, err)
			mockAudit.AssertCalled(t, tt.auditCall, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

// TestProperty_RequestValidationRequiresManagerGroup tests that request validation
// requires both ManagerGroupID and ManagerGroupName to be non-empty
// **Validates: Requirements 1.4**
func TestProperty_RequestValidationRequiresManagerGroup(t *testing.T) {
	ctx := context.Background()

	// Property: For any access request creation attempt, if the ManagerGroupID or
	// ManagerGroupName is empty, then the validation should fail and the request
	// should not be created.

	property := func(managerGroupID, managerGroupName string) bool {
		mockRepo := new(helpers.MockRequestRepository)
		mockValidator := new(helpers.MockRequestValidator)
		mockAuth := new(helpers.MockAuthorizationService)
		mockAudit := new(helpers.MockAuditService)

		svc := service.NewAccessRequestService(mockRepo, mockValidator, mockAuth, mockAudit)

		// Setup valid test data for other fields
		username := "test.user"
		userID := "U123456"
		host := "prod-db-01"
		port := 5432
		accountID := "123456789012"
		expirationDate := time.Now().Add(24 * time.Hour)

		// Mock all validations to pass (we're only testing manager group validation)
		mockValidator.On("ValidateHost", host).Return(&models.ValidationResult{IsValid: true})
		mockValidator.On("ValidatePort", port).Return(&models.ValidationResult{IsValid: true})
		mockValidator.On("ValidateExpirationDate", mock.Anything).Return(&models.ValidationResult{IsValid: true})
		mockValidator.On("ValidateUsername", username).Return(&models.ValidationResult{IsValid: true})
		mockValidator.On("ValidateAccountID", accountID).Return(&models.ValidationResult{IsValid: true})

		// Only mock SaveRequest if both manager group fields are non-empty
		if managerGroupID != "" && managerGroupName != "" {
			mockRepo.On("SaveRequest", ctx, mock.Anything).Return(nil)
			mockAudit.On("LogRequestCreated", ctx, mock.Anything).Return()
		}

		// Attempt to create request
		request, err := svc.CreateRequest(
			ctx,
			username,
			userID,
			host,
			port,
			accountID,
			expirationDate,
			managerGroupID,
			managerGroupName,
			"Test reason",
		)

		// Property check: If either field is empty, creation should fail
		if managerGroupID == "" || managerGroupName == "" {
			return err != nil && request == nil
		}

		// If both fields are non-empty, creation should succeed
		return err == nil && request != nil &&
			request.ManagerGroupID == managerGroupID &&
			request.ManagerGroupName == managerGroupName
	}

	// Test with specific edge cases
	testCases := []struct {
		name             string
		managerGroupID   string
		managerGroupName string
		shouldFail       bool
	}{
		{"Both empty", "", "", true},
		{"ID empty, Name provided", "", "SRE Team", true},
		{"ID provided, Name empty", "G123456", "", true},
		{"Both provided", "G123456", "SRE Team", false},
		{"Both provided with spaces", "G789012", "Cloud Ops Team", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := property(tc.managerGroupID, tc.managerGroupName)
			if !result {
				t.Errorf("Property violated for case: %s (ID=%q, Name=%q)",
					tc.name, tc.managerGroupID, tc.managerGroupName)
			}
		})
	}
}

// TestAccessRequestService_AllowSelfApproval_WhenEnabled tests that self-approval works when explicitly enabled
func TestAccessRequestService_AllowSelfApproval_WhenEnabled(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(helpers.MockRequestRepository)
	mockValidator := new(helpers.MockRequestValidator)
	mockAuth := new(helpers.MockAuthorizationService)
	mockAudit := new(helpers.MockAuditService)

	svc := service.NewAccessRequestService(mockRepo, mockValidator, mockAuth, mockAudit)

	// Enable self-approval for testing
	svc.SetAllowSelfApproval(true)

	// Create a request where the requester will approve their own request
	request := helpers.CreateTestRequest(helpers.ManagerGroupID, "Manager Group")
	requesterID := request.UserID
	requesterName := "self.approver"

	// Mock get request
	mockRepo.On("GetRequestByID", ctx, request.RequestID).Return(request, nil)

	// Mock save request
	mockRepo.On("SaveRequest", ctx, mock.MatchedBy(func(r *models.AccessRequest) bool {
		return r.RequestID == request.RequestID &&
			r.Status == models.RequestStatusPartiallyApproved &&
			r.SecurityApproverID != nil &&
			*r.SecurityApproverID == requesterID
	})).Return(nil)

	// Mock audit logging for security approval (NOT self-approval attempt)
	mockAudit.On("LogSecurityApproval", ctx, requesterID, requesterName, mock.Anything).Return()

	// Execute - approve with same user ID as requester (should succeed when flag is enabled)
	updatedRequest, err := svc.ApproveRequestSecurity(ctx, request.RequestID, requesterID, requesterName)

	// Assert - should succeed
	assert.NoError(t, err)
	assert.NotNil(t, updatedRequest)
	assert.Equal(t, models.RequestStatusPartiallyApproved, updatedRequest.Status)
	assert.True(t, updatedRequest.HasSecurityApproval())
	assert.Equal(t, requesterID, *updatedRequest.SecurityApproverID)

	// Verify self-approval attempt was NOT logged (because it was allowed)
	mockAudit.AssertNotCalled(t, "LogSelfApprovalAttempt")

	// Verify normal approval was logged
	mockAudit.AssertCalled(t, "LogSecurityApproval", ctx, requesterID, requesterName, mock.Anything)

	mockRepo.AssertExpectations(t)
	mockAudit.AssertExpectations(t)
}
