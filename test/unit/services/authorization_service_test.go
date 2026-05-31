package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ssm-access-manager/internal/models"
	"github.com/ssm-access-manager/internal/service"
	"github.com/ssm-access-manager/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestAuthorizationService_IsGroupMember_WithCacheHit tests group membership with cache hit
func TestAuthorizationService_IsGroupMember_WithCacheHit(t *testing.T) {
	ctx := context.Background()
	mockUserRepo := new(helpers.MockUserRepository)
	mockGroupCache := new(helpers.MockGroupMembershipCache)
	mockAudit := new(helpers.MockAuditService)

	svc := service.NewAuthorizationService(mockUserRepo, mockGroupCache, mockAudit)

	groupID := helpers.SecurityGroupID
	userID := helpers.SecurityMemberID

	// Mock cache hit
	mockGroupCache.On("IsMember", ctx, groupID, userID).Return(true, nil)

	// Execute
	isMember, err := svc.IsGroupMember(ctx, groupID, userID)

	// Assert
	assert.NoError(t, err)
	assert.True(t, isMember)
	mockGroupCache.AssertExpectations(t)
}

// TestAuthorizationService_IsGroupMember_WithCacheMiss tests group membership with cache miss
func TestAuthorizationService_IsGroupMember_WithCacheMiss(t *testing.T) {
	ctx := context.Background()
	mockUserRepo := new(helpers.MockUserRepository)
	mockGroupCache := new(helpers.MockGroupMembershipCache)
	mockAudit := new(helpers.MockAuditService)

	svc := service.NewAuthorizationService(mockUserRepo, mockGroupCache, mockAudit)

	groupID := helpers.SecurityGroupID
	userID := helpers.RegularUserID

	// Mock cache miss (user not in group)
	mockGroupCache.On("IsMember", ctx, groupID, userID).Return(false, nil)

	// Execute
	isMember, err := svc.IsGroupMember(ctx, groupID, userID)

	// Assert
	assert.NoError(t, err)
	assert.False(t, isMember)
	mockGroupCache.AssertExpectations(t)
}

// TestAuthorizationService_IsGroupMember_WithError tests error handling
func TestAuthorizationService_IsGroupMember_WithError(t *testing.T) {
	ctx := context.Background()
	mockUserRepo := new(helpers.MockUserRepository)
	mockGroupCache := new(helpers.MockGroupMembershipCache)
	mockAudit := new(helpers.MockAuditService)

	svc := service.NewAuthorizationService(mockUserRepo, mockGroupCache, mockAudit)

	groupID := helpers.SecurityGroupID
	userID := helpers.SecurityMemberID

	// Mock error from cache
	mockGroupCache.On("IsMember", ctx, groupID, userID).Return(false, errors.New("slack API error"))

	// Execute
	isMember, err := svc.IsGroupMember(ctx, groupID, userID)

	// Assert
	assert.Error(t, err)
	assert.False(t, isMember)
	assert.Contains(t, err.Error(), "failed to check group membership")
	mockGroupCache.AssertExpectations(t)
}

// TestAuthorizationService_VerifyAdministratorAuthorization_WithAdmin tests admin verification
func TestAuthorizationService_VerifyAdministratorAuthorization_WithAdmin(t *testing.T) {
	ctx := context.Background()
	mockUserRepo := new(helpers.MockUserRepository)
	mockGroupCache := new(helpers.MockGroupMembershipCache)
	mockAudit := new(helpers.MockAuditService)

	svc := service.NewAuthorizationService(mockUserRepo, mockGroupCache, mockAudit)

	adminUser := helpers.CreateTestAdminUser()

	// Mock user lookup
	mockUserRepo.On("GetUser", ctx, adminUser.UserID).Return(adminUser, nil)

	// Execute
	err := svc.VerifyAdministratorAuthorization(ctx, adminUser.UserID)

	// Assert
	assert.NoError(t, err)
	mockUserRepo.AssertExpectations(t)
}

// TestAuthorizationService_VerifyAdministratorAuthorization_WithNonAdmin tests non-admin rejection
func TestAuthorizationService_VerifyAdministratorAuthorization_WithNonAdmin(t *testing.T) {
	ctx := context.Background()
	mockUserRepo := new(helpers.MockUserRepository)
	mockGroupCache := new(helpers.MockGroupMembershipCache)
	mockAudit := new(helpers.MockAuditService)

	svc := service.NewAuthorizationService(mockUserRepo, mockGroupCache, mockAudit)

	regularUser := helpers.CreateTestRegularUser()

	// Mock user lookup
	mockUserRepo.On("GetUser", ctx, regularUser.UserID).Return(regularUser, nil)

	// Execute
	err := svc.VerifyAdministratorAuthorization(ctx, regularUser.UserID)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
	assert.Contains(t, err.Error(), "not an administrator")
	mockUserRepo.AssertExpectations(t)
}

// TestAuthorizationService_VerifyAdministratorAuthorization_UserNotFound tests missing user
func TestAuthorizationService_VerifyAdministratorAuthorization_UserNotFound(t *testing.T) {
	ctx := context.Background()
	mockUserRepo := new(helpers.MockUserRepository)
	mockGroupCache := new(helpers.MockGroupMembershipCache)
	mockAudit := new(helpers.MockAuditService)

	svc := service.NewAuthorizationService(mockUserRepo, mockGroupCache, mockAudit)

	// Mock user not found
	mockUserRepo.On("GetUser", ctx, "U999999").Return(nil, nil)

	// Execute
	err := svc.VerifyAdministratorAuthorization(ctx, "U999999")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
	mockUserRepo.AssertExpectations(t)
}

// TestAuthorizationService_LogUnauthorizedAttempt tests unauthorized attempt logging
func TestAuthorizationService_LogUnauthorizedAttempt(t *testing.T) {
	ctx := context.Background()
	mockUserRepo := new(helpers.MockUserRepository)
	mockGroupCache := new(helpers.MockGroupMembershipCache)
	mockAudit := new(helpers.MockAuditService)

	svc := service.NewAuthorizationService(mockUserRepo, mockGroupCache, mockAudit)

	userID := "U123456"
	userName := "test.user"
	requestID := "req-123"

	// Mock audit logging
	mockAudit.On("LogUnauthorizedApprovalAttempt", ctx, userID, userName, requestID).Return()

	// Execute
	svc.LogUnauthorizedAttempt(ctx, userID, userName, requestID)

	// Assert
	mockAudit.AssertExpectations(t)
	mockAudit.AssertCalled(t, "LogUnauthorizedApprovalAttempt", ctx, userID, userName, requestID)
}

// TestAuthorizationService_IsAdministrator tests administrator check
func TestAuthorizationService_IsAdministrator(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		user     *models.User
		expected bool
	}{
		{
			name:     "administrator user",
			user:     helpers.CreateTestAdminUser(),
			expected: true,
		},
		{
			name:     "regular user",
			user:     helpers.CreateTestRegularUser(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(helpers.MockUserRepository)
			mockGroupCache := new(helpers.MockGroupMembershipCache)
			mockAudit := new(helpers.MockAuditService)

			svc := service.NewAuthorizationService(mockUserRepo, mockGroupCache, mockAudit)

			// Mock user lookup
			mockUserRepo.On("GetUser", ctx, tt.user.UserID).Return(tt.user, nil)

			// Execute
			isAdmin, err := svc.IsAdministrator(ctx, tt.user.UserID)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, isAdmin)
			mockUserRepo.AssertExpectations(t)
		})
	}
}

// TestAuthorizationService_IsGroupMember_NilCache tests nil cache handling
func TestAuthorizationService_IsGroupMember_NilCache(t *testing.T) {
	ctx := context.Background()
	mockUserRepo := new(helpers.MockUserRepository)
	mockAudit := new(helpers.MockAuditService)

	// Create service with nil cache
	svc := service.NewAuthorizationService(mockUserRepo, nil, mockAudit)

	// Execute
	isMember, err := svc.IsGroupMember(ctx, helpers.SecurityGroupID, helpers.SecurityMemberID)

	// Assert
	assert.Error(t, err)
	assert.False(t, isMember)
	assert.Contains(t, err.Error(), "group cache not initialized")
}

// TestProperty_AdminOperationsFunctionCorrectly tests that admin operations
// maintain correct state and function properly across various scenarios
// **Validates: Requirements 2.6, 6.7**
func TestProperty_AdminOperationsFunctionCorrectly(t *testing.T) {
	ctx := context.Background()

	// Property: For any valid admin operation (add administrator, remove administrator,
	// get administrators, verify administrator authorization), the operation should
	// complete successfully and maintain correct admin user state in the repository.

	testCases := []struct {
		name      string
		operation string
		setup     func(*helpers.MockUserRepository, *helpers.MockGroupMembershipCache, *helpers.MockAuditService)
		execute   func(*service.AuthorizationService) error
		verify    func(*testing.T, error)
	}{
		{
			name:      "Add administrator - new user",
			operation: "AddAdministrator",
			setup: func(mockRepo *helpers.MockUserRepository, mockCache *helpers.MockGroupMembershipCache, mockAudit *helpers.MockAuditService) {
				adminUser := helpers.CreateTestAdminUser()
				// Verify the person adding is an admin
				mockRepo.On("GetUser", ctx, adminUser.UserID).Return(adminUser, nil)
				// Check if user already exists (new user)
				mockRepo.On("GetUser", ctx, "U999999").Return(nil, nil)
				// Save new user
				mockRepo.On("SaveUser", ctx, mock.MatchedBy(func(u *models.User) bool {
					return u.UserID == "U999999" &&
						u.Role == models.UserRoleAdministrator &&
						u.Username == "new.admin" &&
						u.Email == "new.admin@example.com"
				})).Return(nil)
			},
			execute: func(svc *service.AuthorizationService) error {
				adminUser := helpers.CreateTestAdminUser()
				return svc.AddAdministrator(ctx, "U999999", "new.admin", "new.admin@example.com", adminUser.UserID)
			},
			verify: func(t *testing.T, err error) {
				assert.NoError(t, err, "AddAdministrator should succeed for new user")
			},
		},
		{
			name:      "Add administrator - existing user",
			operation: "AddAdministrator",
			setup: func(mockRepo *helpers.MockUserRepository, mockCache *helpers.MockGroupMembershipCache, mockAudit *helpers.MockAuditService) {
				adminUser := helpers.CreateTestAdminUser()
				existingUser := helpers.CreateTestRegularUser()
				// Verify the person adding is an admin
				mockRepo.On("GetUser", ctx, adminUser.UserID).Return(adminUser, nil)
				// Check if user already exists (existing user)
				mockRepo.On("GetUser", ctx, existingUser.UserID).Return(existingUser, nil)
				// Update existing user's role
				mockRepo.On("UpdateUserRole", ctx, existingUser.UserID, models.UserRoleAdministrator).Return(nil)
			},
			execute: func(svc *service.AuthorizationService) error {
				adminUser := helpers.CreateTestAdminUser()
				existingUser := helpers.CreateTestRegularUser()
				return svc.AddAdministrator(ctx, existingUser.UserID, existingUser.Username, existingUser.Email, adminUser.UserID)
			},
			verify: func(t *testing.T, err error) {
				assert.NoError(t, err, "AddAdministrator should succeed for existing user")
			},
		},
		{
			name:      "Add administrator - unauthorized",
			operation: "AddAdministrator",
			setup: func(mockRepo *helpers.MockUserRepository, mockCache *helpers.MockGroupMembershipCache, mockAudit *helpers.MockAuditService) {
				regularUser := helpers.CreateTestRegularUser()
				// Verify the person adding is NOT an admin
				mockRepo.On("GetUser", ctx, regularUser.UserID).Return(regularUser, nil)
			},
			execute: func(svc *service.AuthorizationService) error {
				regularUser := helpers.CreateTestRegularUser()
				return svc.AddAdministrator(ctx, "U999999", "new.admin", "new.admin@example.com", regularUser.UserID)
			},
			verify: func(t *testing.T, err error) {
				assert.Error(t, err, "AddAdministrator should fail for non-admin")
				assert.Contains(t, err.Error(), "unauthorized")
			},
		},
		{
			name:      "Remove administrator - success",
			operation: "RemoveAdministrator",
			setup: func(mockRepo *helpers.MockUserRepository, mockCache *helpers.MockGroupMembershipCache, mockAudit *helpers.MockAuditService) {
				adminUser := helpers.CreateTestAdminUser()
				targetAdmin := &models.User{
					UserID:   "U888888",
					Username: "target.admin",
					Role:     models.UserRoleAdministrator,
					Email:    "target.admin@example.com",
				}
				// Verify the person removing is an admin
				mockRepo.On("GetUser", ctx, adminUser.UserID).Return(adminUser, nil)
				// Get the target user
				mockRepo.On("GetUser", ctx, targetAdmin.UserID).Return(targetAdmin, nil)
				// Update role to regular user
				mockRepo.On("UpdateUserRole", ctx, targetAdmin.UserID, models.UserRoleUser).Return(nil)
			},
			execute: func(svc *service.AuthorizationService) error {
				adminUser := helpers.CreateTestAdminUser()
				return svc.RemoveAdministrator(ctx, "U888888", adminUser.UserID)
			},
			verify: func(t *testing.T, err error) {
				assert.NoError(t, err, "RemoveAdministrator should succeed")
			},
		},
		{
			name:      "Remove administrator - cannot remove self",
			operation: "RemoveAdministrator",
			setup: func(mockRepo *helpers.MockUserRepository, mockCache *helpers.MockGroupMembershipCache, mockAudit *helpers.MockAuditService) {
				adminUser := helpers.CreateTestAdminUser()
				// Verify the person removing is an admin
				mockRepo.On("GetUser", ctx, adminUser.UserID).Return(adminUser, nil)
			},
			execute: func(svc *service.AuthorizationService) error {
				adminUser := helpers.CreateTestAdminUser()
				return svc.RemoveAdministrator(ctx, adminUser.UserID, adminUser.UserID)
			},
			verify: func(t *testing.T, err error) {
				assert.Error(t, err, "RemoveAdministrator should fail when removing self")
				assert.Contains(t, err.Error(), "cannot remove your own administrator privileges")
			},
		},
		{
			name:      "Get all administrators - success",
			operation: "GetAllAdministrators",
			setup: func(mockRepo *helpers.MockUserRepository, mockCache *helpers.MockGroupMembershipCache, mockAudit *helpers.MockAuditService) {
				admins := []*models.User{
					helpers.CreateTestAdminUser(),
					{
						UserID:   "U888888",
						Username: "admin2",
						Role:     models.UserRoleAdministrator,
						Email:    "admin2@example.com",
					},
				}
				mockRepo.On("ListUsersByRole", ctx, models.UserRoleAdministrator).Return(admins, nil)
			},
			execute: func(svc *service.AuthorizationService) error {
				admins, err := svc.GetAllAdministrators(ctx)
				if err != nil {
					return err
				}
				if len(admins) != 2 {
					return errors.New("expected 2 administrators")
				}
				return nil
			},
			verify: func(t *testing.T, err error) {
				assert.NoError(t, err, "GetAllAdministrators should succeed")
			},
		},
		{
			name:      "Verify administrator authorization - admin user",
			operation: "VerifyAdministratorAuthorization",
			setup: func(mockRepo *helpers.MockUserRepository, mockCache *helpers.MockGroupMembershipCache, mockAudit *helpers.MockAuditService) {
				adminUser := helpers.CreateTestAdminUser()
				mockRepo.On("GetUser", ctx, adminUser.UserID).Return(adminUser, nil)
			},
			execute: func(svc *service.AuthorizationService) error {
				adminUser := helpers.CreateTestAdminUser()
				return svc.VerifyAdministratorAuthorization(ctx, adminUser.UserID)
			},
			verify: func(t *testing.T, err error) {
				assert.NoError(t, err, "VerifyAdministratorAuthorization should succeed for admin")
			},
		},
		{
			name:      "Verify administrator authorization - non-admin user",
			operation: "VerifyAdministratorAuthorization",
			setup: func(mockRepo *helpers.MockUserRepository, mockCache *helpers.MockGroupMembershipCache, mockAudit *helpers.MockAuditService) {
				regularUser := helpers.CreateTestRegularUser()
				mockRepo.On("GetUser", ctx, regularUser.UserID).Return(regularUser, nil)
			},
			execute: func(svc *service.AuthorizationService) error {
				regularUser := helpers.CreateTestRegularUser()
				return svc.VerifyAdministratorAuthorization(ctx, regularUser.UserID)
			},
			verify: func(t *testing.T, err error) {
				assert.Error(t, err, "VerifyAdministratorAuthorization should fail for non-admin")
				assert.Contains(t, err.Error(), "unauthorized")
			},
		},
		{
			name:      "Create initial administrator - no existing admins",
			operation: "CreateInitialAdministrator",
			setup: func(mockRepo *helpers.MockUserRepository, mockCache *helpers.MockGroupMembershipCache, mockAudit *helpers.MockAuditService) {
				// No existing administrators
				mockRepo.On("ListUsersByRole", ctx, models.UserRoleAdministrator).Return([]*models.User{}, nil)
				// Save new administrator
				mockRepo.On("SaveUser", ctx, mock.MatchedBy(func(u *models.User) bool {
					return u.UserID == "U111111" &&
						u.Role == models.UserRoleAdministrator &&
						u.Username == "initial.admin" &&
						u.AddedBy == "system"
				})).Return(nil)
			},
			execute: func(svc *service.AuthorizationService) error {
				return svc.CreateInitialAdministrator(ctx, "U111111", "initial.admin", "initial.admin@example.com")
			},
			verify: func(t *testing.T, err error) {
				assert.NoError(t, err, "CreateInitialAdministrator should succeed when no admins exist")
			},
		},
		{
			name:      "Create initial administrator - admins already exist",
			operation: "CreateInitialAdministrator",
			setup: func(mockRepo *helpers.MockUserRepository, mockCache *helpers.MockGroupMembershipCache, mockAudit *helpers.MockAuditService) {
				// Existing administrators
				admins := []*models.User{helpers.CreateTestAdminUser()}
				mockRepo.On("ListUsersByRole", ctx, models.UserRoleAdministrator).Return(admins, nil)
			},
			execute: func(svc *service.AuthorizationService) error {
				return svc.CreateInitialAdministrator(ctx, "U111111", "initial.admin", "initial.admin@example.com")
			},
			verify: func(t *testing.T, err error) {
				assert.Error(t, err, "CreateInitialAdministrator should fail when admins already exist")
				assert.Contains(t, err.Error(), "administrators already exist")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := new(helpers.MockUserRepository)
			mockCache := new(helpers.MockGroupMembershipCache)
			mockAudit := new(helpers.MockAuditService)

			// Setup mocks
			tc.setup(mockRepo, mockCache, mockAudit)

			// Create service
			svc := service.NewAuthorizationService(mockRepo, mockCache, mockAudit)

			// Execute operation
			err := tc.execute(svc)

			// Verify result
			tc.verify(t, err)

			// Verify all expectations were met
			mockRepo.AssertExpectations(t)
		})
	}
}
