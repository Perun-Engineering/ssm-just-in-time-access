package models

// RequestStatus represents the status of an access request
type RequestStatus string

const (
	RequestStatusPending           RequestStatus = "pending"
	RequestStatusPartiallyApproved RequestStatus = "partially_approved"
	RequestStatusApproved          RequestStatus = "approved"
	RequestStatusDenied            RequestStatus = "denied"
	RequestStatusRevoked           RequestStatus = "revoked"
)

// IsValid checks if the request status is valid
func (s RequestStatus) IsValid() bool {
	switch s {
	case RequestStatusPending, RequestStatusPartiallyApproved, RequestStatusApproved, RequestStatusDenied, RequestStatusRevoked:
		return true
	default:
		return false
	}
}

// DocumentStatus represents the status of an SSM document
type DocumentStatus string

const (
	DocumentStatusActive  DocumentStatus = "active"
	DocumentStatusExpired DocumentStatus = "expired"
	DocumentStatusDeleted DocumentStatus = "deleted"
)

// IsValid checks if the document status is valid
func (s DocumentStatus) IsValid() bool {
	switch s {
	case DocumentStatusActive, DocumentStatusExpired, DocumentStatusDeleted:
		return true
	default:
		return false
	}
}

// UserRole represents the role of a user in the system
type UserRole string

const (
	UserRoleUser          UserRole = "user"
	UserRoleAdministrator UserRole = "administrator"
	// REMOVED: UserRoleManager - replaced by Slack user groups
)

// IsValid checks if the user role is valid
func (r UserRole) IsValid() bool {
	switch r {
	case UserRoleUser, UserRoleAdministrator:
		return true
	default:
		return false
	}
}

// AccountStatus represents the status of an AWS account
type AccountStatus string

const (
	AccountStatusActive   AccountStatus = "active"
	AccountStatusInactive AccountStatus = "inactive"
)

// IsValid checks if the account status is valid
func (s AccountStatus) IsValid() bool {
	switch s {
	case AccountStatusActive, AccountStatusInactive:
		return true
	default:
		return false
	}
}
