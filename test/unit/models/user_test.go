package models_test

import (
	"testing"
	"time"

	"github.com/ssm-access-manager/internal/models"
	"github.com/stretchr/testify/assert"
)

// TestUser_Validate_ValidUser tests validation with all required fields
func TestUser_Validate_ValidUser(t *testing.T) {
	now := time.Now()
	user := &models.User{
		UserID:    "U123456",
		Username:  "john.doe",
		Role:      models.UserRoleUser,
		Email:     "john.doe@example.com",
		AddedBy:   "U789012",
		AddedAt:   now,
		UpdatedAt: now,
	}

	err := user.Validate()
	assert.NoError(t, err)
}

// TestUser_Validate_ValidAdministrator tests validation with administrator role
func TestUser_Validate_ValidAdministrator(t *testing.T) {
	now := time.Now()
	user := &models.User{
		UserID:    "U123456",
		Username:  "admin",
		Role:      models.UserRoleAdministrator,
		Email:     "admin@example.com",
		AddedBy:   "U789012",
		AddedAt:   now,
		UpdatedAt: now,
	}

	err := user.Validate()
	assert.NoError(t, err)
}

// TestUser_Validate_MissingUserID tests validation fails without user ID
func TestUser_Validate_MissingUserID(t *testing.T) {
	now := time.Now()
	user := &models.User{
		UserID:    "",
		Username:  "john.doe",
		Role:      models.UserRoleUser,
		Email:     "john.doe@example.com",
		AddedBy:   "U789012",
		AddedAt:   now,
		UpdatedAt: now,
	}

	err := user.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user_id is required")
}

// TestUser_Validate_MissingUsername tests validation fails without username
func TestUser_Validate_MissingUsername(t *testing.T) {
	now := time.Now()
	user := &models.User{
		UserID:    "U123456",
		Username:  "",
		Role:      models.UserRoleUser,
		Email:     "john.doe@example.com",
		AddedBy:   "U789012",
		AddedAt:   now,
		UpdatedAt: now,
	}

	err := user.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username is required")
}

// TestUser_Validate_InvalidRole tests validation fails with invalid role
func TestUser_Validate_InvalidRole(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name string
		role models.UserRole
	}{
		{"empty role", models.UserRole("")},
		{"invalid role", models.UserRole("invalid")},
		{"manager role (removed)", models.UserRole("manager")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &models.User{
				UserID:    "U123456",
				Username:  "john.doe",
				Role:      tt.role,
				Email:     "john.doe@example.com",
				AddedBy:   "U789012",
				AddedAt:   now,
				UpdatedAt: now,
			}

			err := user.Validate()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid role")
		})
	}
}

// TestUser_Validate_MissingEmail tests validation fails without email
func TestUser_Validate_MissingEmail(t *testing.T) {
	now := time.Now()
	user := &models.User{
		UserID:    "U123456",
		Username:  "john.doe",
		Role:      models.UserRoleUser,
		Email:     "",
		AddedBy:   "U789012",
		AddedAt:   now,
		UpdatedAt: now,
	}

	err := user.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "email is required")
}

// TestUser_IsAdministrator tests administrator detection
func TestUser_IsAdministrator(t *testing.T) {
	tests := []struct {
		name     string
		role     models.UserRole
		expected bool
	}{
		{"administrator role", models.UserRoleAdministrator, true},
		{"user role", models.UserRoleUser, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &models.User{
				Role: tt.role,
			}
			result := user.IsAdministrator()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestUserRole_IsValid tests role validation
func TestUserRole_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		role     models.UserRole
		expected bool
	}{
		{"user role is valid", models.UserRoleUser, true},
		{"administrator role is valid", models.UserRoleAdministrator, true},
		{"manager role is invalid", models.UserRole("manager"), false},
		{"empty role is invalid", models.UserRole(""), false},
		{"random role is invalid", models.UserRole("random"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.role.IsValid()
			assert.Equal(t, tt.expected, result)
		})
	}
}
