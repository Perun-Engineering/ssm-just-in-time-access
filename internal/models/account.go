package models

import (
	"fmt"
	"regexp"
	"time"
)

// Account represents an AWS account configuration
type Account struct {
	AccountID     string        `dynamodbav:"account_id" json:"account_id"`
	AccountName   string        `dynamodbav:"account_name" json:"account_name"`
	RoleName      string        `dynamodbav:"role_name" json:"role_name"`
	Regions       []string      `dynamodbav:"regions" json:"regions"`
	BastionHostID string        `dynamodbav:"bastion_host_id,omitempty" json:"bastion_host_id,omitempty"`
	Status        AccountStatus `dynamodbav:"status" json:"status"`
	AddedBy       string        `dynamodbav:"added_by" json:"added_by"`
	AddedAt       time.Time     `dynamodbav:"added_at" json:"added_at"`
	UpdatedAt     time.Time     `dynamodbav:"updated_at" json:"updated_at"`
}

// Validate checks if the account has all required fields
func (a *Account) Validate() error {
	if a.AccountID == "" {
		return fmt.Errorf("account_id is required")
	}
	if a.AccountName == "" {
		return fmt.Errorf("account_name is required")
	}
	if a.RoleName == "" {
		return fmt.Errorf("role_name is required")
	}
	if len(a.Regions) == 0 {
		return fmt.Errorf("at least one region is required")
	}
	if !a.Status.IsValid() {
		return fmt.Errorf("invalid status: %s", a.Status)
	}
	if a.AddedBy == "" {
		return fmt.Errorf("added_by is required")
	}
	if a.AddedAt.IsZero() {
		return fmt.Errorf("added_at is required")
	}
	if a.UpdatedAt.IsZero() {
		return fmt.Errorf("updated_at is required")
	}

	// Validate bastion host ID format if provided
	if a.BastionHostID != "" {
		bastionHostIDPattern := regexp.MustCompile(`^i-[0-9a-f]{8,17}$`)
		if !bastionHostIDPattern.MatchString(a.BastionHostID) {
			return fmt.Errorf("invalid bastion_host_id format: must match pattern i-[0-9a-f]{8,17}")
		}
	}

	return nil
}

// IsActive returns true if the account is active
func (a *Account) IsActive() bool {
	return a.Status == AccountStatusActive
}

// IsInactive returns true if the account is inactive
func (a *Account) IsInactive() bool {
	return a.Status == AccountStatusInactive
}

// HasBastionHost returns true if the account has a bastion host ID configured
func (a *Account) HasBastionHost() bool {
	return a.BastionHostID != ""
}
