package repository

import (
	"context"

	"github.com/ssm-access-manager/internal/models"
)

// ApprovalGroupRepositoryInterface defines the interface for approval group repository operations
type ApprovalGroupRepositoryInterface interface {
	SaveGroup(ctx context.Context, group *models.ApprovalGroup) error
	GetGroup(ctx context.Context, groupID string) (*models.ApprovalGroup, error)
	ListAllGroups(ctx context.Context) ([]*models.ApprovalGroup, error)
	ListGroupsByType(ctx context.Context, groupType models.ApprovalGroupType) ([]*models.ApprovalGroup, error)
	ListActiveManagerGroups(ctx context.Context) ([]*models.ApprovalGroup, error)
	GetSecurityGroup(ctx context.Context) (*models.ApprovalGroup, error)
	UpdateGroup(ctx context.Context, group *models.ApprovalGroup) error
	DeleteGroup(ctx context.Context, groupID string) error
}

// RequestRepositoryInterface defines the interface for request repository operations
type RequestRepositoryInterface interface {
	SaveRequest(ctx context.Context, request *models.AccessRequest) error
	GetRequest(ctx context.Context, requestID string) (*models.AccessRequest, error)
	GetRequestByID(ctx context.Context, requestID string) (*models.AccessRequest, error)
	UpdateRequest(ctx context.Context, request *models.AccessRequest) error
	ListPendingRequests(ctx context.Context) ([]*models.AccessRequest, error)
	ListRequestsByUser(ctx context.Context, userID string) ([]*models.AccessRequest, error)
	ListRequestsByUsername(ctx context.Context, username string) ([]*models.AccessRequest, error)
	ListAllRequests(ctx context.Context) ([]*models.AccessRequest, error)
	DeleteRequest(ctx context.Context, requestID string) error
}

// UserRepositoryInterface defines the interface for user repository operations
type UserRepositoryInterface interface {
	SaveUser(ctx context.Context, user *models.User) error
	GetUser(ctx context.Context, userID string) (*models.User, error)
	ListUsers(ctx context.Context) ([]*models.User, error)
	ListUsersByRole(ctx context.Context, role models.UserRole) ([]*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	UpdateUserRole(ctx context.Context, userID string, role models.UserRole) error
	DeleteUser(ctx context.Context, userID string) error
}

// AccountRepositoryInterface defines the interface for account repository operations
type AccountRepositoryInterface interface {
	SaveAccount(ctx context.Context, account *models.Account) error
	GetAccount(ctx context.Context, accountID string) (*models.Account, error)
	ListAccounts(ctx context.Context) ([]*models.Account, error)
	ListActiveAccounts(ctx context.Context) ([]*models.Account, error)
	UpdateAccount(ctx context.Context, account *models.Account) error
	DeleteAccount(ctx context.Context, accountID string) error
}
