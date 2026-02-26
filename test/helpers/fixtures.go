package helpers

import (
	"time"

	"github.com/google/uuid"
	"github.com/ssm-access-manager/internal/models"
)

// Test user IDs
const (
	SecurityMemberID = "U01SECURITY"
	ManagerMemberID  = "U02MANAGER"
	DualMemberID     = "U03DUAL"
	RegularUserID    = "U04USER"
	AdminUserID      = "U05ADMIN"
)

// Test group IDs
const (
	SecurityGroupID = "S0AFPD6TLQ4"
	ManagerGroupID  = "S0AFYN85MLH"
)

// CreateTestSecurityGroup creates a test security approval group
func CreateTestSecurityGroup() *models.ApprovalGroup {
	return &models.ApprovalGroup{
		GroupID:     SecurityGroupID,
		GroupType:   models.ApprovalGroupTypeSecurity,
		GroupName:   "Security Team",
		SlackHandle: "@ccr-sec",
		Active:      true,
		AddedBy:     AdminUserID,
		AddedAt:     time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// CreateTestManagerGroup creates a test manager approval group
func CreateTestManagerGroup() *models.ApprovalGroup {
	return &models.ApprovalGroup{
		GroupID:     ManagerGroupID,
		GroupType:   models.ApprovalGroupTypeManager,
		GroupName:   "SRE Cloud OPS",
		SlackHandle: "@ccr-cloudops",
		Active:      true,
		AddedBy:     AdminUserID,
		AddedAt:     time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// CreateTestRequest creates a test access request
func CreateTestRequest(managerGroupID, managerGroupName string) *models.AccessRequest {
	return &models.AccessRequest{
		RequestID:        uuid.New().String(),
		UserID:           RegularUserID,
		Username:         "test.user",
		Host:             "test.example.com",
		Port:             5432,
		AccountID:        "123456789012",
		ManagerGroupID:   managerGroupID,
		ManagerGroupName: managerGroupName,
		Reason:           "Test access request reason",
		Status:           models.RequestStatusPending,
		ExpirationDate:   time.Now().Add(14 * 24 * time.Hour),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

// CreateTestLegacyRequest creates a legacy request without manager group
func CreateTestLegacyRequest() *models.AccessRequest {
	return &models.AccessRequest{
		RequestID:      uuid.New().String(),
		UserID:         RegularUserID,
		Username:       "test.user",
		Host:           "test.example.com",
		Port:           22,
		AccountID:      "123456789012",
		Status:         models.RequestStatusPending,
		ExpirationDate: time.Now().Add(14 * 24 * time.Hour),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// CreateTestAdminUser creates a test administrator user
func CreateTestAdminUser() *models.User {
	return &models.User{
		UserID:    AdminUserID,
		Username:  "admin.user",
		Email:     "admin@example.com",
		Role:      models.UserRoleAdministrator,
		AddedBy:   "system",
		AddedAt:   time.Now(),
		UpdatedAt: time.Now(),
	}
}

// CreateTestRegularUser creates a test regular user
func CreateTestRegularUser() *models.User {
	return &models.User{
		UserID:    RegularUserID,
		Username:  "test.user",
		Email:     "test.user@example.com",
		Role:      models.UserRoleUser,
		AddedBy:   AdminUserID,
		AddedAt:   time.Now(),
		UpdatedAt: time.Now(),
	}
}

// CreateTestAccount creates a test AWS account
func CreateTestAccount() *models.Account {
	return &models.Account{
		AccountID:     "123456789012",
		AccountName:   "Test Account",
		RoleName:      "SSMDocumentManagerRole",
		Regions:       []string{"us-east-1"},
		BastionHostID: "i-1234567890abcdef0",
		Status:        models.AccountStatusActive,
		AddedBy:       AdminUserID,
		AddedAt:       time.Now(),
		UpdatedAt:     time.Now(),
	}
}


// CreateSecurityGroup creates a test security approval group
func CreateSecurityGroup() *models.ApprovalGroup {
	now := time.Now()
	return &models.ApprovalGroup{
		GroupID:     "S12345",
		GroupName:   "Security Team",
		GroupType:   models.ApprovalGroupTypeSecurity,
		SlackHandle: "@security",
		Active:      true,
		AddedBy:     "admin-user",
		AddedAt:     now,
		UpdatedAt:   now,
	}
}

// CreateManagerGroup creates a test manager approval group
func CreateManagerGroup(groupID, groupName string) *models.ApprovalGroup {
	now := time.Now()
	return &models.ApprovalGroup{
		GroupID:     groupID,
		GroupName:   groupName,
		GroupType:   models.ApprovalGroupTypeManager,
		SlackHandle: "@" + groupID,
		Active:      true,
		AddedBy:     "admin-user",
		AddedAt:     now,
		UpdatedAt:   now,
	}
}


// CreateAccessRequest creates a simple test access request
func CreateAccessRequest(username, accountID, managerGroupID string) *models.AccessRequest {
	now := time.Now()
	return &models.AccessRequest{
		RequestID:        uuid.New().String(),
		UserID:           "U" + username,
		Username:         username,
		Host:             "test-host.example.com",
		Port:             22,
		AccountID:        accountID,
		ManagerGroupID:   managerGroupID,
		ManagerGroupName: "Manager Group",
		Reason:           "Test access request reason",
		Status:           models.RequestStatusPending,
		ExpirationDate:   now.Add(14 * 24 * time.Hour),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}
