package service

import (
	"context"
	"fmt"
	"time"

	awshelper "github.com/ssm-access-manager/pkg/aws"

	"github.com/ssm-access-manager/internal/models"
	"github.com/ssm-access-manager/internal/repository"
	"github.com/ssm-access-manager/internal/validation"
)

// AccountService handles AWS account management
type AccountService struct {
	accountRepo *repository.AccountRepository
	validator   *validation.RequestValidator
	roleAssumer *awshelper.RoleAssumer
	authService *AuthorizationService
}

// NewAccountService creates a new account service
func NewAccountService(
	accountRepo *repository.AccountRepository,
	validator *validation.RequestValidator,
	roleAssumer *awshelper.RoleAssumer,
	authService *AuthorizationService,
) *AccountService {
	return &AccountService{
		accountRepo: accountRepo,
		validator:   validator,
		roleAssumer: roleAssumer,
		authService: authService,
	}
}

// AddAccount adds a new AWS account to the system
func (s *AccountService) AddAccount(
	ctx context.Context,
	accountID, accountName, roleName string,
	regions []string,
	bastionHostID string,
	addedBy string,
) (*models.Account, error) {
	// Verify administrator authorization
	err := s.authService.VerifyAdministratorAuthorization(ctx, addedBy)
	if err != nil {
		s.authService.LogUnauthorizedAttempt(ctx, addedBy, addedBy, fmt.Sprintf("add_account_%s", accountID))
		return nil, fmt.Errorf("unauthorized: %w", err)
	}

	// Validate account ID
	if result := s.validator.ValidateAccountID(accountID); !result.IsValid {
		return nil, fmt.Errorf("invalid account ID: %s", result.ErrorMessage)
	}

	// Validate role name
	if result := s.validator.ValidateRoleName(roleName); !result.IsValid {
		return nil, fmt.Errorf("invalid role name: %s", result.ErrorMessage)
	}

	// Validate regions
	if len(regions) == 0 {
		return nil, fmt.Errorf("at least one region is required")
	}

	for _, region := range regions {
		if result := s.validator.ValidateRegion(region); !result.IsValid {
			return nil, fmt.Errorf("invalid region %s: %s", region, result.ErrorMessage)
		}
	}

	// Check if account already exists
	existingAccount, err := s.accountRepo.GetAccountByID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing account: %w", err)
	}

	if existingAccount != nil {
		return nil, fmt.Errorf("account %s already exists", accountID)
	}

	// Validate role assumption (test that we can assume the role)
	err = s.roleAssumer.ValidateRoleAssumption(ctx, accountID, roleName)
	if err != nil {
		return nil, fmt.Errorf("failed to validate role assumption: %w", err)
	}

	// Create account
	now := time.Now()
	account := &models.Account{
		AccountID:     accountID,
		AccountName:   accountName,
		RoleName:      roleName,
		Regions:       regions,
		BastionHostID: bastionHostID,
		Status:        models.AccountStatusActive,
		AddedBy:       addedBy,
		AddedAt:       now,
		UpdatedAt:     now,
	}

	// Validate account (includes bastion host ID validation if provided)
	err = account.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid account: %w", err)
	}

	// Save account
	err = s.accountRepo.SaveAccount(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to save account: %w", err)
	}

	return account, nil
}

// RemoveAccount removes an AWS account from the system
func (s *AccountService) RemoveAccount(ctx context.Context, accountID, removedBy string) error {
	// Verify administrator authorization
	err := s.authService.VerifyAdministratorAuthorization(ctx, removedBy)
	if err != nil {
		s.authService.LogUnauthorizedAttempt(ctx, removedBy, removedBy, fmt.Sprintf("remove_account_%s", accountID))
		return fmt.Errorf("unauthorized: %w", err)
	}

	// Check if account exists
	account, err := s.accountRepo.GetAccountByID(ctx, accountID)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}

	if account == nil {
		return fmt.Errorf("account %s not found", accountID)
	}

	// Delete account
	err = s.accountRepo.DeleteAccount(ctx, accountID)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	// Clear cached credentials for this account
	s.roleAssumer.ClearCacheForAccount(accountID)

	return nil
}

// GetAccount retrieves an account by ID
func (s *AccountService) GetAccount(ctx context.Context, accountID string) (*models.Account, error) {
	account, err := s.accountRepo.GetAccountByID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	if account == nil {
		return nil, fmt.Errorf("account %s not found", accountID)
	}

	return account, nil
}

// ListAccounts lists all accounts
func (s *AccountService) ListAccounts(ctx context.Context) ([]*models.Account, error) {
	accounts, err := s.accountRepo.ListAllAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	return accounts, nil
}

// ListActiveAccounts lists all active accounts
func (s *AccountService) ListActiveAccounts(ctx context.Context) ([]*models.Account, error) {
	accounts, err := s.accountRepo.ListActiveAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list active accounts: %w", err)
	}

	return accounts, nil
}

// UpdateAccountStatus updates an account's status
func (s *AccountService) UpdateAccountStatus(
	ctx context.Context,
	accountID string,
	status models.AccountStatus,
	updatedBy string,
) error {
	// Verify administrator authorization
	err := s.authService.VerifyAdministratorAuthorization(ctx, updatedBy)
	if err != nil {
		s.authService.LogUnauthorizedAttempt(ctx, updatedBy, updatedBy, fmt.Sprintf("update_account_status_%s", accountID))
		return fmt.Errorf("unauthorized: %w", err)
	}

	// Validate status
	if !status.IsValid() {
		return fmt.Errorf("invalid status: %s", status)
	}

	// Check if account exists
	account, err := s.accountRepo.GetAccountByID(ctx, accountID)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}

	if account == nil {
		return fmt.Errorf("account %s not found", accountID)
	}

	// Update status
	err = s.accountRepo.UpdateAccountStatus(ctx, accountID, status)
	if err != nil {
		return fmt.Errorf("failed to update account status: %w", err)
	}

	// Clear cached credentials if account is being deactivated
	if status == models.AccountStatusInactive {
		s.roleAssumer.ClearCacheForAccount(accountID)
	}

	return nil
}

// UpdateAccount updates an account's configuration
func (s *AccountService) UpdateAccount(
	ctx context.Context,
	accountID, accountName, roleName string,
	regions []string,
	bastionHostID string,
	updatedBy string,
) (*models.Account, error) {
	// Verify administrator authorization
	err := s.authService.VerifyAdministratorAuthorization(ctx, updatedBy)
	if err != nil {
		s.authService.LogUnauthorizedAttempt(ctx, updatedBy, updatedBy, fmt.Sprintf("update_account_%s", accountID))
		return nil, fmt.Errorf("unauthorized: %w", err)
	}

	// Get existing account
	account, err := s.accountRepo.GetAccountByID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	if account == nil {
		return nil, fmt.Errorf("account %s not found", accountID)
	}

	// Validate role name if changed
	if roleName != account.RoleName {
		if result := s.validator.ValidateRoleName(roleName); !result.IsValid {
			return nil, fmt.Errorf("invalid role name: %s", result.ErrorMessage)
		}

		// Validate role assumption with new role
		err = s.roleAssumer.ValidateRoleAssumption(ctx, accountID, roleName)
		if err != nil {
			return nil, fmt.Errorf("failed to validate role assumption: %w", err)
		}

		// Clear cached credentials since role changed
		s.roleAssumer.ClearCacheForAccount(accountID)
	}

	// Validate regions
	if len(regions) == 0 {
		return nil, fmt.Errorf("at least one region is required")
	}

	for _, region := range regions {
		if result := s.validator.ValidateRegion(region); !result.IsValid {
			return nil, fmt.Errorf("invalid region %s: %s", region, result.ErrorMessage)
		}
	}

	// Update account
	account.AccountName = accountName
	account.RoleName = roleName
	account.Regions = regions
	account.BastionHostID = bastionHostID
	account.UpdatedAt = time.Now()

	// Validate account (includes bastion host ID validation if provided)
	err = account.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid account: %w", err)
	}

	err = s.accountRepo.SaveAccount(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to update account: %w", err)
	}

	return account, nil
}

// ValidateAccountAccess validates that the system can access an account
func (s *AccountService) ValidateAccountAccess(ctx context.Context, accountID string) error {
	account, err := s.accountRepo.GetAccountByID(ctx, accountID)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}

	if account == nil {
		return fmt.Errorf("account %s not found", accountID)
	}

	if !account.IsActive() {
		return fmt.Errorf("account %s is not active", accountID)
	}

	// Validate role assumption
	err = s.roleAssumer.ValidateRoleAssumption(ctx, accountID, account.RoleName)
	if err != nil {
		return fmt.Errorf("failed to validate account access: %w", err)
	}

	return nil
}
