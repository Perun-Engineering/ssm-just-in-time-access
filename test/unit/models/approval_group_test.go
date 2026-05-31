package models_test

import (
	"testing"
	"time"

	"github.com/ssm-access-manager/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestApprovalGroup_Validate_ValidSecurityGroup(t *testing.T) {
	group := &models.ApprovalGroup{
		GroupID:     "S0AFPD6TLQ4",
		GroupType:   models.ApprovalGroupTypeSecurity,
		GroupName:   "Security Team",
		SlackHandle: "@ccr-sec",
		Active:      true,
		AddedBy:     "U05ADMIN",
		AddedAt:     time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := group.Validate()
	assert.NoError(t, err, "Valid security group should pass validation")
}

func TestApprovalGroup_Validate_ValidManagerGroup(t *testing.T) {
	group := &models.ApprovalGroup{
		GroupID:     "S0AFYN85MLH",
		GroupType:   models.ApprovalGroupTypeManager,
		GroupName:   "SRE Cloud OPS",
		SlackHandle: "@ccr-cloudops",
		Active:      true,
		AddedBy:     "U05ADMIN",
		AddedAt:     time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := group.Validate()
	assert.NoError(t, err, "Valid manager group should pass validation")
}

func TestApprovalGroup_Validate_InvalidType(t *testing.T) {
	group := &models.ApprovalGroup{
		GroupID:     "S0AFPD6TLQ4",
		GroupType:   "invalid",
		GroupName:   "Test Group",
		SlackHandle: "@test",
		Active:      true,
		AddedBy:     "U05ADMIN",
		AddedAt:     time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := group.Validate()
	assert.Error(t, err, "Invalid group type should fail validation")
	assert.Contains(t, err.Error(), "type", "Error should mention type")
}

func TestApprovalGroup_Validate_EmptyGroupID(t *testing.T) {
	group := &models.ApprovalGroup{
		GroupID:     "",
		GroupType:   models.ApprovalGroupTypeSecurity,
		GroupName:   "Security Team",
		SlackHandle: "@ccr-sec",
		Active:      true,
		AddedBy:     "U05ADMIN",
		AddedAt:     time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := group.Validate()
	assert.Error(t, err, "Empty group ID should fail validation")
	assert.Contains(t, err.Error(), "group_id", "Error should mention group_id")
}

func TestApprovalGroup_Validate_EmptyGroupName(t *testing.T) {
	group := &models.ApprovalGroup{
		GroupID:     "S0AFPD6TLQ4",
		GroupType:   models.ApprovalGroupTypeSecurity,
		GroupName:   "",
		SlackHandle: "@ccr-sec",
		Active:      true,
		AddedBy:     "U05ADMIN",
		AddedAt:     time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := group.Validate()
	assert.Error(t, err, "Empty group name should fail validation")
	assert.Contains(t, err.Error(), "name", "Error should mention name")
}

func TestApprovalGroup_IsSecurity(t *testing.T) {
	securityGroup := &models.ApprovalGroup{
		GroupType: models.ApprovalGroupTypeSecurity,
	}
	managerGroup := &models.ApprovalGroup{
		GroupType: models.ApprovalGroupTypeManager,
	}

	assert.True(t, securityGroup.IsSecurity(), "Security group should return true for IsSecurity()")
	assert.False(t, managerGroup.IsSecurity(), "Manager group should return false for IsSecurity()")
}

func TestApprovalGroup_IsManager(t *testing.T) {
	securityGroup := &models.ApprovalGroup{
		GroupType: models.ApprovalGroupTypeSecurity,
	}
	managerGroup := &models.ApprovalGroup{
		GroupType: models.ApprovalGroupTypeManager,
	}

	assert.False(t, securityGroup.IsManager(), "Security group should return false for IsManager()")
	assert.True(t, managerGroup.IsManager(), "Manager group should return true for IsManager()")
}

func TestApprovalGroup_IsActive(t *testing.T) {
	activeGroup := &models.ApprovalGroup{
		Active: true,
	}
	inactiveGroup := &models.ApprovalGroup{
		Active: false,
	}

	assert.True(t, activeGroup.IsActive(), "Active group should return true for IsActive()")
	assert.False(t, inactiveGroup.IsActive(), "Inactive group should return false for IsActive()")
}
