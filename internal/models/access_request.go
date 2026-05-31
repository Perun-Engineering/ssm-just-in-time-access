package models

import (
	"fmt"
	"strings"
	"time"
)

// AccessRequest represents a user's request for SSM access
type AccessRequest struct {
	RequestID      string        `dynamodbav:"request_id" json:"request_id"`
	Username       string        `dynamodbav:"username" json:"username"`
	UserID         string        `dynamodbav:"user_id" json:"user_id"`
	Host           string        `dynamodbav:"host" json:"host"`
	Port           int           `dynamodbav:"port" json:"port"`
	AccountID      string        `dynamodbav:"account_id" json:"account_id"`
	ExpirationDate time.Time     `dynamodbav:"expiration_date" json:"expiration_date"`
	Status         RequestStatus `dynamodbav:"status" json:"status"`
	CreatedAt      time.Time     `dynamodbav:"created_at" json:"created_at"`
	UpdatedAt      time.Time     `dynamodbav:"updated_at" json:"updated_at"`

	// Manager group selection (NEW)
	ManagerGroupID   string `dynamodbav:"manager_group_id,omitempty" json:"manager_group_id,omitempty"`
	ManagerGroupName string `dynamodbav:"manager_group_name,omitempty" json:"manager_group_name,omitempty"`

	// Reason for access request
	Reason string `dynamodbav:"reason" json:"reason"`

	// Security approval tracking (NEW)
	SecurityApproverID        *string    `dynamodbav:"security_approver_id,omitempty" json:"security_approver_id,omitempty"`
	SecurityApproverName      *string    `dynamodbav:"security_approver_name,omitempty" json:"security_approver_name,omitempty"`
	SecurityApprovalTimestamp *time.Time `dynamodbav:"security_approval_timestamp,omitempty" json:"security_approval_timestamp,omitempty"`

	// Manager approval tracking (NEW)
	ManagerApproverID        *string    `dynamodbav:"manager_approver_id,omitempty" json:"manager_approver_id,omitempty"`
	ManagerApproverName      *string    `dynamodbav:"manager_approver_name,omitempty" json:"manager_approver_name,omitempty"`
	ManagerApprovalTimestamp *time.Time `dynamodbav:"manager_approval_timestamp,omitempty" json:"manager_approval_timestamp,omitempty"`

	// Legacy fields - kept for backward compatibility with existing DynamoDB records
	// These fields are no longer used in new requests but must remain for data migration
	Approver          *string    `dynamodbav:"approver,omitempty" json:"approver,omitempty"`
	ApproverID        *string    `dynamodbav:"approver_id,omitempty" json:"approver_id,omitempty"`
	ApprovalTimestamp *time.Time `dynamodbav:"approval_timestamp,omitempty" json:"approval_timestamp,omitempty"`

	// Denial and revocation fields
	DenialReason     *string    `dynamodbav:"denial_reason,omitempty" json:"denial_reason,omitempty"`
	RevokedBy        *string    `dynamodbav:"revoked_by,omitempty" json:"revoked_by,omitempty"`
	RevokedByID      *string    `dynamodbav:"revoked_by_id,omitempty" json:"revoked_by_id,omitempty"`
	RevokedAt        *time.Time `dynamodbav:"revoked_at,omitempty" json:"revoked_at,omitempty"`
	RevocationReason *string    `dynamodbav:"revocation_reason,omitempty" json:"revocation_reason,omitempty"`

	// Message metadata for updating approval messages
	// Map of userID -> Slack message timestamp
	ApprovalMessageTimestamps map[string]string `dynamodbav:"approval_message_timestamps,omitempty" json:"approval_message_timestamps,omitempty"`
}

// Validate checks if the access request has all required fields
func (r *AccessRequest) Validate() error {
	if r.RequestID == "" {
		return fmt.Errorf("request_id is required")
	}
	if r.Username == "" {
		return fmt.Errorf("username is required")
	}
	if r.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if r.Host == "" {
		return fmt.Errorf("host is required")
	}
	if r.Port <= 0 || r.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if r.AccountID == "" {
		return fmt.Errorf("account_id is required")
	}
	if r.ExpirationDate.IsZero() {
		return fmt.Errorf("expiration_date is required")
	}
	if !r.Status.IsValid() {
		return fmt.Errorf("invalid status: %s", r.Status)
	}
	if r.CreatedAt.IsZero() {
		return fmt.Errorf("created_at is required")
	}
	if r.UpdatedAt.IsZero() {
		return fmt.Errorf("updated_at is required")
	}
	// Validate reason field
	if strings.TrimSpace(r.Reason) == "" {
		return fmt.Errorf("reason is required")
	}
	return nil
}

// IsPending returns true if the request is pending
func (r *AccessRequest) IsPending() bool {
	return r.Status == RequestStatusPending
}

// IsApproved returns true if the request is approved
func (r *AccessRequest) IsApproved() bool {
	return r.Status == RequestStatusApproved
}

// IsDenied returns true if the request is denied
func (r *AccessRequest) IsDenied() bool {
	return r.Status == RequestStatusDenied
}

// IsRevoked returns true if the request is revoked
func (r *AccessRequest) IsRevoked() bool {
	return r.Status == RequestStatusRevoked
}

// IsPartiallyApproved returns true if the request is partially approved
func (r *AccessRequest) IsPartiallyApproved() bool {
	return r.Status == RequestStatusPartiallyApproved
}

// HasSecurityApproval returns true if security approval has been granted
func (r *AccessRequest) HasSecurityApproval() bool {
	return r.SecurityApproverID != nil
}

// HasManagerApproval returns true if manager approval has been granted
func (r *AccessRequest) HasManagerApproval() bool {
	return r.ManagerApproverID != nil
}

// IsFullyApproved returns true if both security and manager approvals have been granted
func (r *AccessRequest) IsFullyApproved() bool {
	return r.HasSecurityApproval() && r.HasManagerApproval()
}
