package models

import (
	"fmt"
	"time"
)

// SSMDocument represents metadata about an SSM document
type SSMDocument struct {
	DocumentID   string         `dynamodbav:"document_id" json:"document_id"`
	DocumentName string         `dynamodbav:"document_name" json:"document_name"`
	AccountID    string         `dynamodbav:"account_id" json:"account_id"`
	Username     string         `dynamodbav:"username" json:"username"`
	Host         string         `dynamodbav:"host" json:"host"`
	Port         int            `dynamodbav:"port" json:"port"`
	RequestID    string         `dynamodbav:"request_id" json:"request_id"`
	CreatedAt    time.Time      `dynamodbav:"created_at" json:"created_at"`
	ExpiresAt    time.Time      `dynamodbav:"expires_at" json:"expires_at"`
	Status       DocumentStatus `dynamodbav:"status" json:"status"`
	Region       string         `dynamodbav:"region" json:"region"`
	UpdatedAt    time.Time      `dynamodbav:"updated_at" json:"updated_at"`
}

// Validate checks if the SSM document has all required fields
func (d *SSMDocument) Validate() error {
	if d.DocumentID == "" {
		return fmt.Errorf("document_id is required")
	}
	if d.DocumentName == "" {
		return fmt.Errorf("document_name is required")
	}
	if d.AccountID == "" {
		return fmt.Errorf("account_id is required")
	}
	if d.Username == "" {
		return fmt.Errorf("username is required")
	}
	if d.Host == "" {
		return fmt.Errorf("host is required")
	}
	if d.Port <= 0 || d.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if d.RequestID == "" {
		return fmt.Errorf("request_id is required")
	}
	if d.CreatedAt.IsZero() {
		return fmt.Errorf("created_at is required")
	}
	if d.ExpiresAt.IsZero() {
		return fmt.Errorf("expires_at is required")
	}
	if !d.Status.IsValid() {
		return fmt.Errorf("invalid status: %s", d.Status)
	}
	if d.Region == "" {
		return fmt.Errorf("region is required")
	}
	if d.UpdatedAt.IsZero() {
		return fmt.Errorf("updated_at is required")
	}
	return nil
}

// IsActive returns true if the document is active
func (d *SSMDocument) IsActive() bool {
	return d.Status == DocumentStatusActive
}

// IsExpired returns true if the document is expired
func (d *SSMDocument) IsExpired() bool {
	return d.Status == DocumentStatusExpired || time.Now().After(d.ExpiresAt)
}

// IsDeleted returns true if the document is deleted
func (d *SSMDocument) IsDeleted() bool {
	return d.Status == DocumentStatusDeleted
}
