package integration

import (
	"context"
	"testing"
	"time"

	"github.com/ssm-access-manager/internal/models"
	"github.com/ssm-access-manager/internal/service"
	"github.com/ssm-access-manager/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// ApprovalWorkflowTestSuite tests the complete two-tier approval workflow
type ApprovalWorkflowTestSuite struct {
	suite.Suite
	requestService *service.AccessRequestService
	groupService   *service.ApprovalGroupService
	authService    *service.AuthorizationService

	mockRequestRepo  *helpers.MockRequestRepository
	mockGroupRepo    *helpers.MockApprovalGroupRepository
	mockUserRepo     *helpers.MockUserRepository
	mockValidator    *helpers.MockRequestValidator
	mockAuditService *helpers.MockAuditService
	mockCache        *helpers.MockGroupMembershipCache

	securityGroup *models.ApprovalGroup
	managerGroup  *models.ApprovalGroup
	testRequest   *models.AccessRequest
}

// SetupTest runs before each test
func (suite *ApprovalWorkflowTestSuite) SetupTest() {
	// Initialize mocks
	suite.mockRequestRepo = &helpers.MockRequestRepository{}
	suite.mockGroupRepo = &helpers.MockApprovalGroupRepository{}
	suite.mockUserRepo = &helpers.MockUserRepository{}
	suite.mockValidator = &helpers.MockRequestValidator{}
	suite.mockAuditService = &helpers.MockAuditService{}
	suite.mockCache = &helpers.MockGroupMembershipCache{}

	// Create services
	suite.authService = service.NewAuthorizationService(
		suite.mockUserRepo,
		suite.mockCache,
		suite.mockAuditService,
	)

	suite.requestService = service.NewAccessRequestService(
		suite.mockRequestRepo,
		suite.mockValidator,
		suite.authService,
		suite.mockAuditService,
	)

	suite.groupService = service.NewApprovalGroupService(
		suite.mockGroupRepo,
		suite.authService,
		suite.mockAuditService,
	)

	// Setup test data
	suite.securityGroup = helpers.CreateSecurityGroup()
	suite.managerGroup = helpers.CreateManagerGroup("S0AFYN85MLH", "SRE Cloud OPS")
	suite.testRequest = helpers.CreateTestRequest(suite.managerGroup.GroupID, suite.managerGroup.GroupName)
}

// TestTwoTierApprovalWorkflow_SecurityFirst tests security approval followed by manager approval
func (suite *ApprovalWorkflowTestSuite) TestTwoTierApprovalWorkflow_SecurityFirst() {
	ctx := context.Background()

	// Setup: Request exists in pending state
	suite.mockRequestRepo.On("GetRequestByID", ctx, suite.testRequest.RequestID).Return(suite.testRequest, nil)
	suite.mockRequestRepo.On("SaveRequest", ctx, suite.testRequest).Return(nil)
	suite.mockAuditService.On("LogSecurityApproval", ctx, "U01SECURITY", "Security User", suite.testRequest).Return()
	suite.mockAuditService.On("LogManagerApproval", ctx, "U02MANAGER", "Manager User", suite.testRequest).Return()

	// Step 1: Security member approves
	_, err := suite.requestService.ApproveRequestSecurity(ctx, suite.testRequest.RequestID, "U01SECURITY", "Security User")
	assert.NoError(suite.T(), err)

	// Assert: Status should be partially_approved
	assert.Equal(suite.T(), models.RequestStatusPartiallyApproved, suite.testRequest.Status)
	assert.True(suite.T(), suite.testRequest.HasSecurityApproval())
	assert.False(suite.T(), suite.testRequest.HasManagerApproval())
	assert.False(suite.T(), suite.testRequest.IsFullyApproved())

	// Step 2: Manager member approves
	_, err = suite.requestService.ApproveRequestManager(ctx, suite.testRequest.RequestID, "U02MANAGER", "Manager User")
	assert.NoError(suite.T(), err)

	// Assert: Status should be approved
	assert.Equal(suite.T(), models.RequestStatusApproved, suite.testRequest.Status)
	assert.True(suite.T(), suite.testRequest.HasSecurityApproval())
	assert.True(suite.T(), suite.testRequest.HasManagerApproval())
	assert.True(suite.T(), suite.testRequest.IsFullyApproved())

	// Verify all mocks were called
	suite.mockRequestRepo.AssertExpectations(suite.T())
	suite.mockAuditService.AssertExpectations(suite.T())
}

// TestTwoTierApprovalWorkflow_ManagerFirst tests manager approval followed by security approval
func (suite *ApprovalWorkflowTestSuite) TestTwoTierApprovalWorkflow_ManagerFirst() {
	ctx := context.Background()

	// Setup: Request exists in pending state
	suite.mockRequestRepo.On("GetRequestByID", ctx, suite.testRequest.RequestID).Return(suite.testRequest, nil)
	suite.mockRequestRepo.On("SaveRequest", ctx, suite.testRequest).Return(nil)
	suite.mockAuditService.On("LogManagerApproval", ctx, "U02MANAGER", "Manager User", suite.testRequest).Return()
	suite.mockAuditService.On("LogSecurityApproval", ctx, "U01SECURITY", "Security User", suite.testRequest).Return()

	// Step 1: Manager member approves first
	_, err := suite.requestService.ApproveRequestManager(ctx, suite.testRequest.RequestID, "U02MANAGER", "Manager User")
	assert.NoError(suite.T(), err)

	// Assert: Status should be partially_approved
	assert.Equal(suite.T(), models.RequestStatusPartiallyApproved, suite.testRequest.Status)
	assert.False(suite.T(), suite.testRequest.HasSecurityApproval())
	assert.True(suite.T(), suite.testRequest.HasManagerApproval())
	assert.False(suite.T(), suite.testRequest.IsFullyApproved())

	// Step 2: Security member approves
	_, err = suite.requestService.ApproveRequestSecurity(ctx, suite.testRequest.RequestID, "U01SECURITY", "Security User")
	assert.NoError(suite.T(), err)

	// Assert: Status should be approved (order doesn't matter)
	assert.Equal(suite.T(), models.RequestStatusApproved, suite.testRequest.Status)
	assert.True(suite.T(), suite.testRequest.HasSecurityApproval())
	assert.True(suite.T(), suite.testRequest.HasManagerApproval())
	assert.True(suite.T(), suite.testRequest.IsFullyApproved())

	// Verify all mocks were called
	suite.mockRequestRepo.AssertExpectations(suite.T())
	suite.mockAuditService.AssertExpectations(suite.T())
}

// TestDuplicateApprovalPrevention tests that the same approval type cannot be granted twice
func (suite *ApprovalWorkflowTestSuite) TestDuplicateApprovalPrevention() {
	ctx := context.Background()

	// Setup: Request with security approval already granted
	securityApproverID := "U01SECURITY"
	securityApproverName := "Security User"
	now := time.Now()
	suite.testRequest.SecurityApproverID = &securityApproverID
	suite.testRequest.SecurityApproverName = &securityApproverName
	suite.testRequest.SecurityApprovalTimestamp = &now
	suite.testRequest.Status = models.RequestStatusPartiallyApproved

	suite.mockRequestRepo.On("GetRequestByID", ctx, suite.testRequest.RequestID).Return(suite.testRequest, nil)

	// Attempt: Another security member tries to approve
	_, err := suite.requestService.ApproveRequestSecurity(ctx, suite.testRequest.RequestID, "U03SECURITY2", "Security User 2")

	// Assert: Should return error
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "security approval already granted")

	// Verify status unchanged
	assert.Equal(suite.T(), models.RequestStatusPartiallyApproved, suite.testRequest.Status)

	suite.mockRequestRepo.AssertExpectations(suite.T())
}

// TestDenialFromSecurityMember tests that a security member can deny a request
func (suite *ApprovalWorkflowTestSuite) TestDenialFromSecurityMember() {
	ctx := context.Background()
	reason := "Security concerns identified"

	// Setup: Request exists in pending state
	suite.mockRequestRepo.On("GetRequestByID", ctx, suite.testRequest.RequestID).Return(suite.testRequest, nil)
	suite.mockRequestRepo.On("SaveRequest", ctx, suite.testRequest).Return(nil)
	suite.mockAuditService.On("LogRequestDenied", ctx, "U01SECURITY", "Security User", suite.testRequest, reason).Return()

	// Action: Security member denies
	_, err := suite.requestService.DenyRequest(ctx, suite.testRequest.RequestID, "U01SECURITY", "Security User", reason)
	assert.NoError(suite.T(), err)

	// Assert: Status should be denied
	assert.Equal(suite.T(), models.RequestStatusDenied, suite.testRequest.Status)
	assert.Equal(suite.T(), "U01SECURITY", *suite.testRequest.ApproverID)
	assert.Equal(suite.T(), "Security User", *suite.testRequest.Approver)
	assert.Equal(suite.T(), reason, *suite.testRequest.DenialReason)

	suite.mockRequestRepo.AssertExpectations(suite.T())
	suite.mockAuditService.AssertExpectations(suite.T())
}

// TestDenialFromManagerMember tests that a manager member can deny a request
func (suite *ApprovalWorkflowTestSuite) TestDenialFromManagerMember() {
	ctx := context.Background()
	reason := "Resource not available"

	// Setup: Request exists in pending state
	suite.mockRequestRepo.On("GetRequestByID", ctx, suite.testRequest.RequestID).Return(suite.testRequest, nil)
	suite.mockRequestRepo.On("SaveRequest", ctx, suite.testRequest).Return(nil)
	suite.mockAuditService.On("LogRequestDenied", ctx, "U02MANAGER", "Manager User", suite.testRequest, reason).Return()

	// Action: Manager member denies
	_, err := suite.requestService.DenyRequest(ctx, suite.testRequest.RequestID, "U02MANAGER", "Manager User", reason)
	assert.NoError(suite.T(), err)

	// Assert: Status should be denied
	assert.Equal(suite.T(), models.RequestStatusDenied, suite.testRequest.Status)
	assert.Equal(suite.T(), "U02MANAGER", *suite.testRequest.ApproverID)
	assert.Equal(suite.T(), "Manager User", *suite.testRequest.Approver)
	assert.Equal(suite.T(), reason, *suite.testRequest.DenialReason)

	suite.mockRequestRepo.AssertExpectations(suite.T())
	suite.mockAuditService.AssertExpectations(suite.T())
}

// TestPartialApprovalCanBeDenied tests that a partially approved request can be denied
func (suite *ApprovalWorkflowTestSuite) TestPartialApprovalCanBeDenied() {
	ctx := context.Background()

	// Setup: Request with security approval already granted
	securityApproverID := "U01SECURITY"
	securityApproverName := "Security User"
	suite.testRequest.SecurityApproverID = &securityApproverID
	suite.testRequest.SecurityApproverName = &securityApproverName
	suite.testRequest.Status = models.RequestStatusPartiallyApproved

	suite.mockRequestRepo.On("GetRequestByID", ctx, suite.testRequest.RequestID).Return(suite.testRequest, nil)
	suite.mockRequestRepo.On("SaveRequest", ctx, mock.Anything).Return(nil)
	suite.mockAuditService.On("LogRequestDenied", ctx, "U02MANAGER", "Manager User", mock.Anything, "Changed mind").Return()

	// Attempt: Manager denies after security approved
	updatedRequest, err := suite.requestService.DenyRequest(ctx, suite.testRequest.RequestID, "U02MANAGER", "Manager User", "Changed mind")

	// Assert: Should succeed
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), updatedRequest)
	assert.Equal(suite.T(), models.RequestStatusDenied, updatedRequest.Status)
	assert.Equal(suite.T(), "Changed mind", *updatedRequest.DenialReason)

	suite.mockRequestRepo.AssertExpectations(suite.T())
	suite.mockAuditService.AssertExpectations(suite.T())
}

// TestApprovalWorkflowTestSuite runs the test suite
func TestApprovalWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(ApprovalWorkflowTestSuite))
}
