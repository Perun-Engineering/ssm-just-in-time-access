package models

import (
	"fmt"
	"time"
)

// ApprovalGroupType represents the type of approval group
type ApprovalGroupType string

const (
	ApprovalGroupTypeSecurity ApprovalGroupType = "security"
	ApprovalGroupTypeManager  ApprovalGroupType = "manager"
)

// IsValid checks if the approval group type is valid
func (t ApprovalGroupType) IsValid() bool {
	return t == ApprovalGroupTypeSecurity || t == ApprovalGroupTypeManager
}

// ApprovalGroup represents a Slack user group used for approvals
type ApprovalGroup struct {
	GroupID     string            `dynamodbav:"group_id" json:"group_id"`         // Slack user group ID (partition key)
	GroupName   string            `dynamodbav:"group_name" json:"group_name"`     // Display name
	GroupType   ApprovalGroupType `dynamodbav:"group_type" json:"group_type"`     // "security" or "manager"
	SlackHandle string            `dynamodbav:"slack_handle" json:"slack_handle"` // e.g., "@ccr-sec"
	Active      bool              `dynamodbav:"active" json:"active"`             // Whether group is active
	AddedBy     string            `dynamodbav:"added_by" json:"added_by"`         // Admin who added
	AddedAt     time.Time         `dynamodbav:"added_at" json:"added_at"`
	UpdatedAt   time.Time         `dynamodbav:"updated_at" json:"updated_at"`
}

// Validate checks if the approval group has all required fields
func (g *ApprovalGroup) Validate() error {
	if g.GroupID == "" {
		return fmt.Errorf("group_id is required")
	}
	if g.GroupName == "" {
		return fmt.Errorf("group_name is required")
	}
	if !g.GroupType.IsValid() {
		return fmt.Errorf("invalid group_type: %s (must be 'security' or 'manager')", g.GroupType)
	}
	if g.SlackHandle == "" {
		return fmt.Errorf("slack_handle is required")
	}
	if g.AddedBy == "" {
		return fmt.Errorf("added_by is required")
	}
	if g.AddedAt.IsZero() {
		return fmt.Errorf("added_at is required")
	}
	if g.UpdatedAt.IsZero() {
		return fmt.Errorf("updated_at is required")
	}
	return nil
}

// IsSecurity returns true if the group is a security group
func (g *ApprovalGroup) IsSecurity() bool {
	return g.GroupType == ApprovalGroupTypeSecurity
}

// IsManager returns true if the group is a manager group
func (g *ApprovalGroup) IsManager() bool {
	return g.GroupType == ApprovalGroupTypeManager
}

// IsActive returns true if the group is active
func (g *ApprovalGroup) IsActive() bool {
	return g.Active
}
