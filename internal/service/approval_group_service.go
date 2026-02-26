package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ssm-access-manager/internal/models"
	"github.com/ssm-access-manager/internal/repository"
)

// ApprovalGroupService handles business logic for approval groups
type ApprovalGroupService struct {
	repo         repository.ApprovalGroupRepositoryInterface
	authService  AuthorizationServiceInterface
	auditService AuditServiceInterface
}

// NewApprovalGroupService creates a new approval group service
func NewApprovalGroupService(
	repo repository.ApprovalGroupRepositoryInterface,
	authService AuthorizationServiceInterface,
	auditService AuditServiceInterface,
) *ApprovalGroupService {
	return &ApprovalGroupService{
		repo:         repo,
		authService:  authService,
		auditService: auditService,
	}
}

// AddGroup adds a new approval group
func (s *ApprovalGroupService) AddGroup(ctx context.Context, group *models.ApprovalGroup, adminID, adminName string) error {
	// Verify administrator authorization
	err := s.authService.VerifyAdministratorAuthorization(ctx, adminID)
	if err != nil {
		s.authService.LogUnauthorizedAttempt(ctx, adminID, adminName, fmt.Sprintf("add_approval_group_%s", group.GroupID))
		return fmt.Errorf("unauthorized: only administrators can add approval groups: %w", err)
	}

	// Validate the group
	if err := group.Validate(); err != nil {
		return fmt.Errorf("invalid approval group: %w", err)
	}

	// If this is a security group, check if one already exists
	if group.IsSecurity() {
		existingSecurityGroup, err := s.repo.GetSecurityGroup(ctx)
		if err == nil && existingSecurityGroup != nil {
			return fmt.Errorf("security group already exists: %s", existingSecurityGroup.GroupID)
		}
	}

	// Set timestamps
	now := time.Now()
	group.AddedAt = now
	group.UpdatedAt = now
	group.AddedBy = adminID

	// Save the group
	err = s.repo.SaveGroup(ctx, group)
	if err != nil {
		return fmt.Errorf("failed to save approval group: %w", err)
	}

	// Log the action
	if s.auditService != nil {
		s.auditService.LogApprovalGroupAdded(ctx, adminID, adminName, group)
	}

	return nil
}


// UpdateGroup updates an existing approval group
func (s *ApprovalGroupService) UpdateGroup(ctx context.Context, groupID string, updates map[string]interface{}, adminID, adminName string) error {
	// Verify administrator authorization
	err := s.authService.VerifyAdministratorAuthorization(ctx, adminID)
	if err != nil {
		s.authService.LogUnauthorizedAttempt(ctx, adminID, adminName, fmt.Sprintf("update_approval_group_%s", groupID))
		return fmt.Errorf("unauthorized: only administrators can update approval groups: %w", err)
	}

	// Get the existing group
	group, err := s.repo.GetGroup(ctx, groupID)
	if err != nil {
		return fmt.Errorf("failed to get approval group: %w", err)
	}

	// Apply updates
	if name, ok := updates["name"].(string); ok && name != "" {
		group.GroupName = name
	}
	if active, ok := updates["active"].(bool); ok {
		group.Active = active
	}

	// Update timestamp
	group.UpdatedAt = time.Now()

	// Save the updated group
	err = s.repo.UpdateGroup(ctx, group)
	if err != nil {
		return fmt.Errorf("failed to update approval group: %w", err)
	}

	// Log the action
	if s.auditService != nil {
		s.auditService.LogApprovalGroupUpdated(ctx, adminID, adminName, group)
	}

	return nil
}

// RemoveGroup removes an approval group
func (s *ApprovalGroupService) RemoveGroup(ctx context.Context, groupID, adminID, adminName string) error {
	// Verify administrator authorization
	err := s.authService.VerifyAdministratorAuthorization(ctx, adminID)
	if err != nil {
		s.authService.LogUnauthorizedAttempt(ctx, adminID, adminName, fmt.Sprintf("remove_approval_group_%s", groupID))
		return fmt.Errorf("unauthorized: only administrators can remove approval groups: %w", err)
	}

	// Check if group exists
	group, err := s.repo.GetGroup(ctx, groupID)
	if err != nil {
		return fmt.Errorf("failed to get approval group: %w", err)
	}

	// Don't allow removing the security group if it's the only one
	if group.IsSecurity() {
		return fmt.Errorf("cannot remove security group: at least one security group is required")
	}

	// Delete the group
	err = s.repo.DeleteGroup(ctx, groupID)
	if err != nil {
		return fmt.Errorf("failed to delete approval group: %w", err)
	}

	// Log the action
	if s.auditService != nil {
		s.auditService.LogApprovalGroupRemoved(ctx, adminID, adminName, groupID)
	}

	return nil
}

// GetGroup retrieves an approval group by ID
func (s *ApprovalGroupService) GetGroup(ctx context.Context, groupID string) (*models.ApprovalGroup, error) {
	group, err := s.repo.GetGroup(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get approval group: %w", err)
	}
	return group, nil
}

// ListAllGroups retrieves all approval groups
func (s *ApprovalGroupService) ListAllGroups(ctx context.Context) ([]*models.ApprovalGroup, error) {
	groups, err := s.repo.ListAllGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list approval groups: %w", err)
	}
	return groups, nil
}

// GetSecurityGroup retrieves the security group
func (s *ApprovalGroupService) GetSecurityGroup(ctx context.Context) (*models.ApprovalGroup, error) {
	group, err := s.repo.GetSecurityGroup(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get security group: %w", err)
	}
	return group, nil
}

// ListActiveManagerGroups retrieves all active manager groups
func (s *ApprovalGroupService) ListActiveManagerGroups(ctx context.Context) ([]*models.ApprovalGroup, error) {
	groups, err := s.repo.ListActiveManagerGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list active manager groups: %w", err)
	}
	return groups, nil
}
