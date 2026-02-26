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

// TestApprovalGroupService_AddGroup_WithAdminAuthorization tests adding a group as admin
func TestApprovalGroupService_AddGroup_WithAdminAuthorization(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(helpers.MockApprovalGroupRepository)
	mockAuth := new(helpers.MockAuthorizationService)
	mockAudit := new(helpers.MockAuditService)

	svc := service.NewApprovalGroupService(mockRepo, mockAuth, mockAudit)

	group := helpers.CreateTestManagerGroup()
	adminID := "U123456"
	adminName := "admin"

	// Mock admin authorization
	mockAuth.On("VerifyAdministratorAuthorization", ctx, adminID).Return(nil)
	
	// Mock save operation
	mockRepo.On("SaveGroup", ctx, mock.MatchedBy(func(g *models.ApprovalGroup) bool {
		return g.GroupID == group.GroupID && !g.AddedAt.IsZero() && !g.UpdatedAt.IsZero()
	})).Return(nil)
	
	// Mock audit logging
	mockAudit.On("LogApprovalGroupAdded", ctx, adminID, adminName, mock.Anything).Return()

	err := svc.AddGroup(ctx, group, adminID, adminName)
	
	assert.NoError(t, err)
	assert.NotZero(t, group.AddedAt)
	assert.NotZero(t, group.UpdatedAt)
	assert.Equal(t, adminID, group.AddedBy)
	mockAuth.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockAudit.AssertExpectations(t)
}

// TestApprovalGroupService_AddGroup_RejectsNonAdmin tests non-admin cannot add groups
func TestApprovalGroupService_AddGroup_RejectsNonAdmin(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(helpers.MockApprovalGroupRepository)
	mockAuth := new(helpers.MockAuthorizationService)
	mockAudit := new(helpers.MockAuditService)

	svc := service.NewApprovalGroupService(mockRepo, mockAuth, mockAudit)

	group := helpers.CreateTestManagerGroup()
	userID := "U999999"
	userName := "regular.user"

	// Mock authorization failure
	mockAuth.On("VerifyAdministratorAuthorization", ctx, userID).Return(errors.New("not an administrator"))
	mockAuth.On("LogUnauthorizedAttempt", ctx, userID, userName, mock.MatchedBy(func(action string) bool {
		return action == "add_approval_group_"+group.GroupID
	})).Return()

	err := svc.AddGroup(ctx, group, userID, userName)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
	mockAuth.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "SaveGroup")
	mockAudit.AssertNotCalled(t, "LogApprovalGroupAdded")
}

// TestApprovalGroupService_AddGroup_ValidatesGroupData tests validation is enforced
func TestApprovalGroupService_AddGroup_ValidatesGroupData(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(helpers.MockApprovalGroupRepository)
	mockAuth := new(helpers.MockAuthorizationService)
	mockAudit := new(helpers.MockAuditService)

	svc := service.NewApprovalGroupService(mockRepo, mockAuth, mockAudit)

	// Create invalid group (missing required fields)
	group := &models.ApprovalGroup{
		GroupID: "", // Invalid: empty
	}
	adminID := "U123456"
	adminName := "admin"

	mockAuth.On("VerifyAdministratorAuthorization", ctx, adminID).Return(nil)

	err := svc.AddGroup(ctx, group, adminID, adminName)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid approval group")
	mockRepo.AssertNotCalled(t, "SaveGroup")
	mockAudit.AssertNotCalled(t, "LogApprovalGroupAdded")
}

// TestApprovalGroupService_AddGroup_EnforcesSingleSecurityGroup tests only one security group allowed
func TestApprovalGroupService_AddGroup_EnforcesSingleSecurityGroup(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(helpers.MockApprovalGroupRepository)
	mockAuth := new(helpers.MockAuthorizationService)
	mockAudit := new(helpers.MockAuditService)

	svc := service.NewApprovalGroupService(mockRepo, mockAuth, mockAudit)

	newSecurityGroup := helpers.CreateTestSecurityGroup()
	newSecurityGroup.GroupID = "S999999" // Different ID
	adminID := "U123456"
	adminName := "admin"

	mockAuth.On("VerifyAdministratorAuthorization", ctx, adminID).Return(nil)
	
	// Mock existing security group
	existingGroup := helpers.CreateTestSecurityGroup()
	mockRepo.On("GetSecurityGroup", ctx).Return(existingGroup, nil)

	err := svc.AddGroup(ctx, newSecurityGroup, adminID, adminName)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "security group already exists")
	mockRepo.AssertNotCalled(t, "SaveGroup")
	mockAudit.AssertNotCalled(t, "LogApprovalGroupAdded")
}

// TestApprovalGroupService_UpdateGroup_WithValidChanges tests updating a group
func TestApprovalGroupService_UpdateGroup_WithValidChanges(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(helpers.MockApprovalGroupRepository)
	mockAuth := new(helpers.MockAuthorizationService)
	mockAudit := new(helpers.MockAuditService)

	svc := service.NewApprovalGroupService(mockRepo, mockAuth, mockAudit)

	existingGroup := helpers.CreateTestManagerGroup()
	groupID := existingGroup.GroupID
	adminID := "U123456"
	adminName := "admin"
	
	updates := map[string]interface{}{
		"name":   "Updated Manager Group",
		"active": false,
	}

	mockAuth.On("VerifyAdministratorAuthorization", ctx, adminID).Return(nil)
	mockRepo.On("GetGroup", ctx, groupID).Return(existingGroup, nil)
	mockRepo.On("UpdateGroup", ctx, mock.MatchedBy(func(g *models.ApprovalGroup) bool {
		return g.GroupName == "Updated Manager Group" && g.Active == false
	})).Return(nil)
	mockAudit.On("LogApprovalGroupUpdated", ctx, adminID, adminName, mock.Anything).Return()

	err := svc.UpdateGroup(ctx, groupID, updates, adminID, adminName)
	
	assert.NoError(t, err)
	mockAuth.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockAudit.AssertExpectations(t)
}

// TestApprovalGroupService_RemoveGroup_WithAuthorization tests removing a manager group
func TestApprovalGroupService_RemoveGroup_WithAuthorization(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(helpers.MockApprovalGroupRepository)
	mockAuth := new(helpers.MockAuthorizationService)
	mockAudit := new(helpers.MockAuditService)

	svc := service.NewApprovalGroupService(mockRepo, mockAuth, mockAudit)

	managerGroup := helpers.CreateTestManagerGroup()
	groupID := managerGroup.GroupID
	adminID := "U123456"
	adminName := "admin"

	mockAuth.On("VerifyAdministratorAuthorization", ctx, adminID).Return(nil)
	mockRepo.On("GetGroup", ctx, groupID).Return(managerGroup, nil)
	mockRepo.On("DeleteGroup", ctx, groupID).Return(nil)
	mockAudit.On("LogApprovalGroupRemoved", ctx, adminID, adminName, groupID).Return()

	err := svc.RemoveGroup(ctx, groupID, adminID, adminName)
	
	assert.NoError(t, err)
	mockAuth.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockAudit.AssertExpectations(t)
}

// TestApprovalGroupService_RemoveGroup_CannotRemoveSecurityGroup tests security group protection
func TestApprovalGroupService_RemoveGroup_CannotRemoveSecurityGroup(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(helpers.MockApprovalGroupRepository)
	mockAuth := new(helpers.MockAuthorizationService)
	mockAudit := new(helpers.MockAuditService)

	svc := service.NewApprovalGroupService(mockRepo, mockAuth, mockAudit)

	securityGroup := helpers.CreateTestSecurityGroup()
	groupID := securityGroup.GroupID
	adminID := "U123456"
	adminName := "admin"

	mockAuth.On("VerifyAdministratorAuthorization", ctx, adminID).Return(nil)
	mockRepo.On("GetGroup", ctx, groupID).Return(securityGroup, nil)

	err := svc.RemoveGroup(ctx, groupID, adminID, adminName)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot remove security group")
	mockRepo.AssertNotCalled(t, "DeleteGroup")
	mockAudit.AssertNotCalled(t, "LogApprovalGroupRemoved")
}

// TestApprovalGroupService_GetSecurityGroup tests retrieving security group
func TestApprovalGroupService_GetSecurityGroup(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(helpers.MockApprovalGroupRepository)
	mockAuth := new(helpers.MockAuthorizationService)
	mockAudit := new(helpers.MockAuditService)

	svc := service.NewApprovalGroupService(mockRepo, mockAuth, mockAudit)

	expectedGroup := helpers.CreateTestSecurityGroup()
	mockRepo.On("GetSecurityGroup", ctx).Return(expectedGroup, nil)

	group, err := svc.GetSecurityGroup(ctx)
	
	assert.NoError(t, err)
	assert.Equal(t, expectedGroup.GroupID, group.GroupID)
	assert.True(t, group.IsSecurity())
	mockRepo.AssertExpectations(t)
}

// TestApprovalGroupService_ListActiveManagerGroups tests filtering active manager groups
func TestApprovalGroupService_ListActiveManagerGroups(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(helpers.MockApprovalGroupRepository)
	mockAuth := new(helpers.MockAuthorizationService)
	mockAudit := new(helpers.MockAuditService)

	svc := service.NewApprovalGroupService(mockRepo, mockAuth, mockAudit)

	expectedGroups := []*models.ApprovalGroup{
		helpers.CreateTestManagerGroup(),
	}
	mockRepo.On("ListActiveManagerGroups", ctx).Return(expectedGroups, nil)

	groups, err := svc.ListActiveManagerGroups(ctx)
	
	assert.NoError(t, err)
	assert.Len(t, groups, 1)
	assert.True(t, groups[0].IsManager())
	assert.True(t, groups[0].Active)
	mockRepo.AssertExpectations(t)
}

// TestApprovalGroupService_AuditLogging tests all operations are logged
func TestApprovalGroupService_AuditLogging(t *testing.T) {
	ctx := context.Background()
	
	tests := []struct {
		name      string
		operation func(*service.ApprovalGroupService, *helpers.MockApprovalGroupRepository, *helpers.MockAuthorizationService, *helpers.MockAuditService) error
		auditCall string
	}{
		{
			name: "AddGroup logs",
			operation: func(svc *service.ApprovalGroupService, repo *helpers.MockApprovalGroupRepository, auth *helpers.MockAuthorizationService, audit *helpers.MockAuditService) error {
				group := helpers.CreateTestManagerGroup()
				auth.On("VerifyAdministratorAuthorization", ctx, "U123456").Return(nil)
				repo.On("SaveGroup", ctx, mock.Anything).Return(nil)
				audit.On("LogApprovalGroupAdded", ctx, "U123456", "admin", mock.Anything).Return()
				return svc.AddGroup(ctx, group, "U123456", "admin")
			},
			auditCall: "LogApprovalGroupAdded",
		},
		{
			name: "UpdateGroup logs",
			operation: func(svc *service.ApprovalGroupService, repo *helpers.MockApprovalGroupRepository, auth *helpers.MockAuthorizationService, audit *helpers.MockAuditService) error {
				group := helpers.CreateTestManagerGroup()
				auth.On("VerifyAdministratorAuthorization", ctx, "U123456").Return(nil)
				repo.On("GetGroup", ctx, group.GroupID).Return(group, nil)
				repo.On("UpdateGroup", ctx, mock.Anything).Return(nil)
				audit.On("LogApprovalGroupUpdated", ctx, "U123456", "admin", mock.Anything).Return()
				return svc.UpdateGroup(ctx, group.GroupID, map[string]interface{}{"name": "New Name"}, "U123456", "admin")
			},
			auditCall: "LogApprovalGroupUpdated",
		},
		{
			name: "RemoveGroup logs",
			operation: func(svc *service.ApprovalGroupService, repo *helpers.MockApprovalGroupRepository, auth *helpers.MockAuthorizationService, audit *helpers.MockAuditService) error {
				group := helpers.CreateTestManagerGroup()
				auth.On("VerifyAdministratorAuthorization", ctx, "U123456").Return(nil)
				repo.On("GetGroup", ctx, group.GroupID).Return(group, nil)
				repo.On("DeleteGroup", ctx, group.GroupID).Return(nil)
				audit.On("LogApprovalGroupRemoved", ctx, "U123456", "admin", group.GroupID).Return()
				return svc.RemoveGroup(ctx, group.GroupID, "U123456", "admin")
			},
			auditCall: "LogApprovalGroupRemoved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(helpers.MockApprovalGroupRepository)
			mockAuth := new(helpers.MockAuthorizationService)
			mockAudit := new(helpers.MockAuditService)
			svc := service.NewApprovalGroupService(mockRepo, mockAuth, mockAudit)

			err := tt.operation(svc, mockRepo, mockAuth, mockAudit)
			
			assert.NoError(t, err)
			mockAudit.AssertCalled(t, tt.auditCall, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}
