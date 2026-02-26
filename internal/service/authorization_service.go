package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ssm-access-manager/internal/models"
	"github.com/ssm-access-manager/internal/repository"
)

// AuthorizationService handles user authorization and role management
type AuthorizationService struct {
	userRepo     repository.UserRepositoryInterface
	groupCache   GroupMembershipCacheInterface
	auditService AuditServiceInterface
}

// NewAuthorizationService creates a new authorization service
func NewAuthorizationService(
	userRepo repository.UserRepositoryInterface,
	groupCache GroupMembershipCacheInterface,
	auditService AuditServiceInterface,
) *AuthorizationService {
	return &AuthorizationService{
		userRepo:     userRepo,
		groupCache:   groupCache,
		auditService: auditService,
	}
}


// IsGroupMember checks if a user is a member of a Slack user group
func (s *AuthorizationService) IsGroupMember(ctx context.Context, groupID, userID string) (bool, error) {
	if s.groupCache == nil {
		return false, fmt.Errorf("group cache not initialized")
	}

	isMember, err := s.groupCache.IsMember(ctx, groupID, userID)
	if err != nil {
		return false, fmt.Errorf("failed to check group membership: %w", err)
	}

	return isMember, nil
}

// IsAdministrator checks if a user is an administrator
func (s *AuthorizationService) IsAdministrator(ctx context.Context, userID string) (bool, error) {
	user, err := s.userRepo.GetUser(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return false, nil
	}

	return user.IsAdministrator(), nil
}



// AddAdministrator adds a user as an administrator
func (s *AuthorizationService) AddAdministrator(ctx context.Context, userID, username, email, addedBy string) error {
	// Verify the person adding is an administrator
	isAdmin, err := s.IsAdministrator(ctx, addedBy)
	if err != nil {
		return fmt.Errorf("failed to verify administrator: %w", err)
	}
	if !isAdmin {
		return fmt.Errorf("unauthorized: only administrators can add administrators")
	}

	// Check if user already exists
	existingUser, err := s.userRepo.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to check existing user: %w", err)
	}

	now := time.Now()

	if existingUser != nil {
		// Update existing user's role
		err = s.userRepo.UpdateUserRole(ctx, userID, models.UserRoleAdministrator)
		if err != nil {
			return fmt.Errorf("failed to update user role: %w", err)
		}
		return nil
	}

	// Create new user with administrator role
	user := &models.User{
		UserID:    userID,
		Username:  username,
		Role:      models.UserRoleAdministrator,
		Email:     email,
		AddedBy:   addedBy,
		AddedAt:   now,
		UpdatedAt: now,
	}

	err = s.userRepo.SaveUser(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to save administrator: %w", err)
	}

	return nil
}

// RemoveAdministrator removes a user from the administrator role
func (s *AuthorizationService) RemoveAdministrator(ctx context.Context, userID, removedBy string) error {
	// Verify the person removing is an administrator
	isAdmin, err := s.IsAdministrator(ctx, removedBy)
	if err != nil {
		return fmt.Errorf("failed to verify administrator: %w", err)
	}
	if !isAdmin {
		return fmt.Errorf("unauthorized: only administrators can remove administrators")
	}

	// Prevent removing yourself
	if userID == removedBy {
		return fmt.Errorf("cannot remove your own administrator privileges")
	}

	// Get the user
	user, err := s.userRepo.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found: %s", userID)
	}

	// Update role to regular user
	err = s.userRepo.UpdateUserRole(ctx, userID, models.UserRoleUser)
	if err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}

	return nil
}


// GetAllAdministrators retrieves all users with administrator role
func (s *AuthorizationService) GetAllAdministrators(ctx context.Context) ([]*models.User, error) {
	admins, err := s.userRepo.ListUsersByRole(ctx, models.UserRoleAdministrator)
	if err != nil {
		return nil, fmt.Errorf("failed to list administrators: %w", err)
	}
	return admins, nil
}


// VerifyAdministratorAuthorization verifies that a user is authorized to perform admin actions
func (s *AuthorizationService) VerifyAdministratorAuthorization(ctx context.Context, userID string) error {
	isAdmin, err := s.IsAdministrator(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to verify administrator authorization: %w", err)
	}

	if !isAdmin {
		return fmt.Errorf("unauthorized: user %s is not an administrator", userID)
	}

	return nil
}

// LogUnauthorizedAttempt logs an unauthorized access attempt
// LogUnauthorizedAttempt logs an unauthorized access attempt
func (s *AuthorizationService) LogUnauthorizedAttempt(ctx context.Context, userID, userName, requestID string) {
	if s.auditService != nil {
		s.auditService.LogUnauthorizedApprovalAttempt(ctx, userID, userName, requestID)
	}
}

// CreateInitialAdministrator creates the first administrator (bootstrap)
// This should only be called during initial setup
func (s *AuthorizationService) CreateInitialAdministrator(ctx context.Context, userID, username, email string) error {
	// Check if any administrators exist
	admins, err := s.GetAllAdministrators(ctx)
	if err != nil {
		return fmt.Errorf("failed to check existing administrators: %w", err)
	}

	if len(admins) > 0 {
		return fmt.Errorf("administrators already exist, cannot create initial administrator")
	}

	// Create the initial administrator
	now := time.Now()
	user := &models.User{
		UserID:    userID,
		Username:  username,
		Role:      models.UserRoleAdministrator,
		Email:     email,
		AddedBy:   "system",
		AddedAt:   now,
		UpdatedAt: now,
	}

	err = s.userRepo.SaveUser(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to create initial administrator: %w", err)
	}

	return nil
}
