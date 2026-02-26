package models

import (
	"fmt"
	"time"
)

// User represents a user in the system
type User struct {
	UserID    string    `dynamodbav:"user_id" json:"user_id"`
	Username  string    `dynamodbav:"username" json:"username"`
	Role      UserRole  `dynamodbav:"role" json:"role"`
	Email     string    `dynamodbav:"email" json:"email"`
	AddedBy   string    `dynamodbav:"added_by" json:"added_by"`
	AddedAt   time.Time `dynamodbav:"added_at" json:"added_at"`
	UpdatedAt time.Time `dynamodbav:"updated_at" json:"updated_at"`
}

// Validate checks if the user has all required fields
func (u *User) Validate() error {
	if u.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if u.Username == "" {
		return fmt.Errorf("username is required")
	}
	if !u.Role.IsValid() {
		return fmt.Errorf("invalid role: %s", u.Role)
	}
	if u.Email == "" {
		return fmt.Errorf("email is required")
	}
	if u.AddedBy == "" {
		return fmt.Errorf("added_by is required")
	}
	if u.AddedAt.IsZero() {
		return fmt.Errorf("added_at is required")
	}
	if u.UpdatedAt.IsZero() {
		return fmt.Errorf("updated_at is required")
	}
	return nil
}

// IsAdministrator returns true if the user is an administrator
func (u *User) IsAdministrator() bool {
	return u.Role == UserRoleAdministrator
}
