package models_test

import (
	"testing"
	"time"

	"github.com/ssm-access-manager/internal/models"
	"github.com/stretchr/testify/assert"
)

// TestAccessRequest_Validate_ValidRequest tests validation with all required fields
func TestAccessRequest_Validate_ValidRequest(t *testing.T) {
	now := time.Now()
	request := &models.AccessRequest{
		RequestID:      "req-123",
		Username:       "john.doe",
		UserID:         "U123456",
		Host:           "prod-db-01",
		Port:           5432,
		AccountID:      "123456789012",
		ExpirationDate: now.Add(24 * time.Hour),
		Status:         models.RequestStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
		ManagerGroupID: "S123456",
		Reason:         "Need to investigate production database issue",
	}

	err := request.Validate()
	assert.NoError(t, err)
}

// TestAccessRequest_Validate_MissingRequestID tests validation fails without request ID
func TestAccessRequest_Validate_MissingRequestID(t *testing.T) {
	now := time.Now()
	request := &models.AccessRequest{
		RequestID:      "",
		Username:       "john.doe",
		UserID:         "U123456",
		Host:           "prod-db-01",
		Port:           5432,
		AccountID:      "123456789012",
		ExpirationDate: now.Add(24 * time.Hour),
		Status:         models.RequestStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err := request.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request_id is required")
}

// TestAccessRequest_Validate_MissingUsername tests validation fails without username
func TestAccessRequest_Validate_MissingUsername(t *testing.T) {
	now := time.Now()
	request := &models.AccessRequest{
		RequestID:      "req-123",
		Username:       "",
		UserID:         "U123456",
		Host:           "prod-db-01",
		Port:           5432,
		AccountID:      "123456789012",
		ExpirationDate: now.Add(24 * time.Hour),
		Status:         models.RequestStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err := request.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username is required")
}

// TestAccessRequest_Validate_InvalidPort tests validation fails with invalid port
func TestAccessRequest_Validate_InvalidPort(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name string
		port int
	}{
		{"zero port", 0},
		{"negative port", -1},
		{"port too high", 65536},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &models.AccessRequest{
				RequestID:      "req-123",
				Username:       "john.doe",
				UserID:         "U123456",
				Host:           "prod-db-01",
				Port:           tt.port,
				AccountID:      "123456789012",
				ExpirationDate: now.Add(24 * time.Hour),
				Status:         models.RequestStatusPending,
				CreatedAt:      now,
				UpdatedAt:      now,
			}

			err := request.Validate()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "port must be between 1 and 65535")
		})
	}
}

// TestAccessRequest_Validate_InvalidStatus tests validation fails with invalid status
func TestAccessRequest_Validate_InvalidStatus(t *testing.T) {
	now := time.Now()
	request := &models.AccessRequest{
		RequestID:      "req-123",
		Username:       "john.doe",
		UserID:         "U123456",
		Host:           "prod-db-01",
		Port:           5432,
		AccountID:      "123456789012",
		ExpirationDate: now.Add(24 * time.Hour),
		Status:         models.RequestStatus("invalid"),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err := request.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status")
}

// TestAccessRequest_HasSecurityApproval tests security approval detection
func TestAccessRequest_HasSecurityApproval(t *testing.T) {
	approverID := "U123456"
	
	tests := []struct {
		name     string
		request  *models.AccessRequest
		expected bool
	}{
		{
			name: "has security approval",
			request: &models.AccessRequest{
				SecurityApproverID: &approverID,
			},
			expected: true,
		},
		{
			name: "no security approval",
			request: &models.AccessRequest{
				SecurityApproverID: nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.request.HasSecurityApproval()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAccessRequest_HasManagerApproval tests manager approval detection
func TestAccessRequest_HasManagerApproval(t *testing.T) {
	approverID := "U789012"
	
	tests := []struct {
		name     string
		request  *models.AccessRequest
		expected bool
	}{
		{
			name: "has manager approval",
			request: &models.AccessRequest{
				ManagerApproverID: &approverID,
			},
			expected: true,
		},
		{
			name: "no manager approval",
			request: &models.AccessRequest{
				ManagerApproverID: nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.request.HasManagerApproval()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAccessRequest_IsFullyApproved tests full approval logic
func TestAccessRequest_IsFullyApproved(t *testing.T) {
	securityApproverID := "U123456"
	managerApproverID := "U789012"
	
	tests := []struct {
		name     string
		request  *models.AccessRequest
		expected bool
	}{
		{
			name: "both approvals granted",
			request: &models.AccessRequest{
				SecurityApproverID: &securityApproverID,
				ManagerApproverID:  &managerApproverID,
			},
			expected: true,
		},
		{
			name: "only security approval",
			request: &models.AccessRequest{
				SecurityApproverID: &securityApproverID,
				ManagerApproverID:  nil,
			},
			expected: false,
		},
		{
			name: "only manager approval",
			request: &models.AccessRequest{
				SecurityApproverID: nil,
				ManagerApproverID:  &managerApproverID,
			},
			expected: false,
		},
		{
			name: "no approvals",
			request: &models.AccessRequest{
				SecurityApproverID: nil,
				ManagerApproverID:  nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.request.IsFullyApproved()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAccessRequest_IsPartiallyApproved tests partial approval status
func TestAccessRequest_IsPartiallyApproved(t *testing.T) {
	tests := []struct {
		name     string
		status   models.RequestStatus
		expected bool
	}{
		{"partially approved status", models.RequestStatusPartiallyApproved, true},
		{"pending status", models.RequestStatusPending, false},
		{"approved status", models.RequestStatusApproved, false},
		{"denied status", models.RequestStatusDenied, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &models.AccessRequest{
				Status: tt.status,
			}
			result := request.IsPartiallyApproved()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAccessRequest_StatusTransitions tests status helper methods
func TestAccessRequest_StatusTransitions(t *testing.T) {
	tests := []struct {
		name       string
		status     models.RequestStatus
		isPending  bool
		isApproved bool
		isDenied   bool
		isRevoked  bool
	}{
		{
			name:       "pending status",
			status:     models.RequestStatusPending,
			isPending:  true,
			isApproved: false,
			isDenied:   false,
			isRevoked:  false,
		},
		{
			name:       "approved status",
			status:     models.RequestStatusApproved,
			isPending:  false,
			isApproved: true,
			isDenied:   false,
			isRevoked:  false,
		},
		{
			name:       "denied status",
			status:     models.RequestStatusDenied,
			isPending:  false,
			isApproved: false,
			isDenied:   true,
			isRevoked:  false,
		},
		{
			name:       "revoked status",
			status:     models.RequestStatusRevoked,
			isPending:  false,
			isApproved: false,
			isDenied:   false,
			isRevoked:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &models.AccessRequest{
				Status: tt.status,
			}
			
			assert.Equal(t, tt.isPending, request.IsPending(), "IsPending mismatch")
			assert.Equal(t, tt.isApproved, request.IsApproved(), "IsApproved mismatch")
			assert.Equal(t, tt.isDenied, request.IsDenied(), "IsDenied mismatch")
			assert.Equal(t, tt.isRevoked, request.IsRevoked(), "IsRevoked mismatch")
		})
	}
}

// TestAccessRequest_Validate_EmptyReason tests validation fails with empty reason
func TestAccessRequest_Validate_EmptyReason(t *testing.T) {
	now := time.Now()
	request := &models.AccessRequest{
		RequestID:      "req-123",
		Username:       "john.doe",
		UserID:         "U123456",
		Host:           "prod-db-01",
		Port:           5432,
		AccountID:      "123456789012",
		ExpirationDate: now.Add(24 * time.Hour),
		Status:         models.RequestStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
		Reason:         "",
	}

	err := request.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reason is required")
}

// TestAccessRequest_Validate_WhitespaceOnlyReason tests validation fails with whitespace-only reason
func TestAccessRequest_Validate_WhitespaceOnlyReason(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name   string
		reason string
	}{
		{"single space", " "},
		{"multiple spaces", "   "},
		{"tab character", "\t"},
		{"newline character", "\n"},
		{"mixed whitespace", " \t\n "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &models.AccessRequest{
				RequestID:      "req-123",
				Username:       "john.doe",
				UserID:         "U123456",
				Host:           "prod-db-01",
				Port:           5432,
				AccountID:      "123456789012",
				ExpirationDate: now.Add(24 * time.Hour),
				Status:         models.RequestStatusPending,
				CreatedAt:      now,
				UpdatedAt:      now,
				Reason:         tt.reason,
			}

			err := request.Validate()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "reason is required")
		})
	}
}

// TestAccessRequest_Validate_ValidReason tests validation succeeds with valid reason
func TestAccessRequest_Validate_ValidReason(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name   string
		reason string
	}{
		{"simple reason", "Database investigation"},
		{"detailed reason", "Need to investigate production database issue #1234"},
		{"reason with leading/trailing spaces", "  Valid reason with spaces  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &models.AccessRequest{
				RequestID:      "req-123",
				Username:       "john.doe",
				UserID:         "U123456",
				Host:           "prod-db-01",
				Port:           5432,
				AccountID:      "123456789012",
				ExpirationDate: now.Add(24 * time.Hour),
				Status:         models.RequestStatusPending,
				CreatedAt:      now,
				UpdatedAt:      now,
				Reason:         tt.reason,
			}

			err := request.Validate()
			assert.NoError(t, err)
		})
	}
}
