package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ssm-access-manager/internal/models"
)

// AssertRequestStatus asserts that a request has the expected status
func AssertRequestStatus(t *testing.T, request *models.AccessRequest, expectedStatus models.RequestStatus) {
	t.Helper()
	assert.Equal(t, expectedStatus, request.Status, "Request status should be %s", expectedStatus)
}

// AssertRequestPartiallyApproved asserts that a request is partially approved
func AssertRequestPartiallyApproved(t *testing.T, request *models.AccessRequest) {
	t.Helper()
	assert.Equal(t, models.RequestStatusPartiallyApproved, request.Status, "Request should be partially approved")
	assert.True(t, request.IsPartiallyApproved(), "IsPartiallyApproved() should return true")
}

// AssertRequestFullyApproved asserts that a request is fully approved
func AssertRequestFullyApproved(t *testing.T, request *models.AccessRequest) {
	t.Helper()
	assert.Equal(t, models.RequestStatusApproved, request.Status, "Request should be fully approved")
	assert.True(t, request.IsFullyApproved(), "IsFullyApproved() should return true")
	assert.True(t, request.HasSecurityApproval(), "Should have security approval")
	assert.True(t, request.HasManagerApproval(), "Should have manager approval")
}

// AssertSecurityApprovalGranted asserts that security approval was granted
func AssertSecurityApprovalGranted(t *testing.T, request *models.AccessRequest, approverID, approverName string) {
	t.Helper()
	assert.True(t, request.HasSecurityApproval(), "Should have security approval")
	assert.Equal(t, approverID, request.SecurityApproverID, "Security approver ID should match")
	assert.Equal(t, approverName, request.SecurityApproverName, "Security approver name should match")
	assert.NotNil(t, request.SecurityApprovalTimestamp, "Security approval timestamp should be set")
}

// AssertManagerApprovalGranted asserts that manager approval was granted
func AssertManagerApprovalGranted(t *testing.T, request *models.AccessRequest, approverID, approverName string) {
	t.Helper()
	assert.True(t, request.HasManagerApproval(), "Should have manager approval")
	assert.Equal(t, approverID, request.ManagerApproverID, "Manager approver ID should match")
	assert.Equal(t, approverName, request.ManagerApproverName, "Manager approver name should match")
	assert.NotNil(t, request.ManagerApprovalTimestamp, "Manager approval timestamp should be set")
}

// AssertApprovalGroupValid asserts that an approval group is valid
func AssertApprovalGroupValid(t *testing.T, group *models.ApprovalGroup) {
	t.Helper()
	assert.NoError(t, group.Validate(), "Approval group should be valid")
	assert.NotEmpty(t, group.GroupID, "Group ID should not be empty")
	assert.NotEmpty(t, group.GroupName, "Group name should not be empty")
	assert.True(t, group.GroupType == models.ApprovalGroupTypeSecurity || group.GroupType == models.ApprovalGroupTypeManager, "Group type should be security or manager")
}

// AssertRequestIsLegacy asserts that a request is a legacy request
func AssertRequestIsLegacy(t *testing.T, request *models.AccessRequest) {
	t.Helper()
	assert.Empty(t, request.ManagerGroupID, "Legacy request should not have manager group ID")
	assert.Empty(t, request.ManagerGroupName, "Legacy request should not have manager group name")
}

// AssertRequestIsTwoTier asserts that a request uses two-tier approval
func AssertRequestIsTwoTier(t *testing.T, request *models.AccessRequest) {
	t.Helper()
	assert.NotEmpty(t, request.ManagerGroupID, "Two-tier request should have manager group ID")
	assert.NotEmpty(t, request.ManagerGroupName, "Two-tier request should have manager group name")
}
