package service

import (
	"context"
	"time"

	"github.com/ssm-access-manager/internal/models"
)

// AuthorizationServiceInterface defines the interface for authorization operations
type AuthorizationServiceInterface interface {
	VerifyAdministratorAuthorization(ctx context.Context, userID string) error
	IsGroupMember(ctx context.Context, groupID, userID string) (bool, error)
	LogUnauthorizedAttempt(ctx context.Context, userID, userName, action string)
}

// AuditServiceInterface defines the interface for audit logging operations
type AuditServiceInterface interface {
	LogApprovalGroupAdded(ctx context.Context, adminID, adminName string, group interface{})
	LogApprovalGroupUpdated(ctx context.Context, adminID, adminName string, group interface{})
	LogApprovalGroupRemoved(ctx context.Context, adminID, adminName, groupID string)
	LogRequestCreated(ctx context.Context, request *models.AccessRequest)
	LogSecurityApproval(ctx context.Context, approverID, approverName string, request *models.AccessRequest)
	LogManagerApproval(ctx context.Context, approverID, approverName string, request *models.AccessRequest)
	LogRequestDenied(ctx context.Context, denierID, denierName string, request *models.AccessRequest, reason string)
	LogRequestRevoked(ctx context.Context, revokerID, revokerName string, request *models.AccessRequest, reason string)
	LogUnauthorizedApprovalAttempt(ctx context.Context, userID, userName, requestID string)
	LogSelfApprovalAttempt(ctx context.Context, userID, username, requestID string)
}

// RequestValidatorInterface defines the interface for request validation
type RequestValidatorInterface interface {
	ValidateHost(host string) *models.ValidationResult
	ValidatePort(port int) *models.ValidationResult
	ValidateExpirationDate(expirationDate time.Time) *models.ValidationResult
	ValidateUsername(username string) *models.ValidationResult
	ValidateAccountID(accountID string) *models.ValidationResult
}


// GroupMembershipCacheInterface defines the interface for group membership caching
type GroupMembershipCacheInterface interface {
	IsMember(ctx context.Context, groupID, userID string) (bool, error)
	GetMembers(ctx context.Context, groupID string) ([]string, error)
}
