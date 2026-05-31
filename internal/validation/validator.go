package validation

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/ssm-access-manager/internal/models"
)

// RequestValidator validates access request parameters
type RequestValidator struct {
	maxExpirationDays int
}

// NewRequestValidator creates a new request validator
func NewRequestValidator(maxExpirationDays int) *RequestValidator {
	if maxExpirationDays <= 0 {
		maxExpirationDays = 90 // Default to 90 days
	}
	return &RequestValidator{
		maxExpirationDays: maxExpirationDays,
	}
}

// ValidateHost validates a hostname or IP address
func (v *RequestValidator) ValidateHost(host string) *models.ValidationResult {
	if host == "" {
		result := models.Invalid("host is required")
		return &result
	}

	// Store original for debugging
	originalHost := host

	// Trim whitespace
	host = strings.TrimSpace(host)

	// Debug: log if trimming changed anything
	if originalHost != host {
		// Host had whitespace - this could indicate parsing issues
		_ = originalHost // Keep for potential logging
	}

	// Check if it's a valid IP address
	if ip := net.ParseIP(host); ip != nil {
		result := models.Valid()
		return &result
	}

	// Check if it's a valid hostname (RFC 1123)
	// Hostname can contain alphanumeric characters, hyphens, and dots
	// Each label (part between dots) must:
	// - Be 1-63 characters long
	// - Start and end with alphanumeric
	// - Can contain hyphens in the middle
	// Pattern: label is either single char OR starts with alphanumeric, has 0+ middle chars, ends with alphanumeric
	hostnameRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?)*$`)

	if !hostnameRegex.MatchString(host) {
		result := models.Invalid("invalid host format: must be a valid hostname or IP address")
		return &result
	}

	// Check length (max 253 characters for FQDN)
	if len(host) > 253 {
		result := models.Invalid("host is too long: maximum 253 characters")
		return &result
	}

	// Validate each label is max 63 characters
	labels := strings.Split(host, ".")
	for _, label := range labels {
		if len(label) > 63 {
			result := models.Invalid("hostname label is too long: maximum 63 characters per label")
			return &result
		}
	}

	result := models.Valid()
	return &result
}

// ValidatePort validates a port number
func (v *RequestValidator) ValidatePort(port int) *models.ValidationResult {
	if port < 1 || port > 65535 {
		result := models.Invalid(fmt.Sprintf("invalid port: must be between 1 and 65535, got %d", port))
		return &result
	}
	result := models.Valid()
	return &result
}

// ValidateExpirationDate validates an expiration date
func (v *RequestValidator) ValidateExpirationDate(date time.Time) *models.ValidationResult {
	now := time.Now()

	// Check if date is in the past
	if date.Before(now) {
		result := models.Invalid("expiration date must be in the future")
		return &result
	}

	// Check if date is too far in the future
	maxDate := now.AddDate(0, 0, v.maxExpirationDays)
	if date.After(maxDate) {
		result := models.Invalid(fmt.Sprintf("expiration date must be within %d days from now", v.maxExpirationDays))
		return &result
	}

	result := models.Valid()
	return &result
}

// ValidateUsername validates a username
func (v *RequestValidator) ValidateUsername(username string) *models.ValidationResult {
	if username == "" {
		result := models.Invalid("username is required")
		return &result
	}

	// Trim whitespace
	username = strings.TrimSpace(username)

	if username == "" {
		result := models.Invalid("username cannot be empty or whitespace only")
		return &result
	}

	// Username should be reasonable length
	if len(username) > 64 {
		result := models.Invalid("username is too long: maximum 64 characters")
		return &result
	}

	result := models.Valid()
	return &result
}

// ValidateAccountID validates an AWS account ID
func (v *RequestValidator) ValidateAccountID(accountID string) *models.ValidationResult {
	if accountID == "" {
		result := models.Invalid("account_id is required")
		return &result
	}

	// AWS account IDs are exactly 12 digits
	accountIDRegex := regexp.MustCompile(`^\d{12}$`)
	if !accountIDRegex.MatchString(accountID) {
		result := models.Invalid("invalid account_id: must be a 12-digit number")
		return &result
	}

	result := models.Valid()
	return &result
}

// ValidateEmail validates an email address
func (v *RequestValidator) ValidateEmail(email string) *models.ValidationResult {
	if email == "" {
		result := models.Invalid("email is required")
		return &result
	}

	// Simple email validation regex
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		result := models.Invalid("invalid email format")
		return &result
	}

	result := models.Valid()
	return &result
}

// ValidateRegion validates an AWS region
func (v *RequestValidator) ValidateRegion(region string) *models.ValidationResult {
	if region == "" {
		result := models.Invalid("region is required")
		return &result
	}

	// AWS region format: us-east-1, eu-west-2, etc.
	regionRegex := regexp.MustCompile(`^[a-z]{2}-[a-z]+-\d+$`)
	if !regionRegex.MatchString(region) {
		result := models.Invalid("invalid region format")
		return &result
	}

	result := models.Valid()
	return &result
}

// ValidateRoleName validates an IAM role name
func (v *RequestValidator) ValidateRoleName(roleName string) *models.ValidationResult {
	if roleName == "" {
		result := models.Invalid("role_name is required")
		return &result
	}

	// IAM role names can contain alphanumeric characters, plus (+), equals (=), comma (,), period (.), at (@), underscore (_), and hyphen (-)
	roleNameRegex := regexp.MustCompile(`^[\w+=,.@\-]+$`)
	if !roleNameRegex.MatchString(roleName) {
		result := models.Invalid("invalid role_name: can only contain alphanumeric characters and +=,.@_-")
		return &result
	}

	if len(roleName) > 64 {
		result := models.Invalid("role_name is too long: maximum 64 characters")
		return &result
	}

	result := models.Valid()
	return &result
}

// SanitizeForDocumentName sanitizes a string for use in SSM document names
// AWS SSM document names can only contain alphanumeric characters, hyphens, and underscores
func (v *RequestValidator) SanitizeForDocumentName(value string) string {
	// Replace any character that's not alphanumeric, hyphen, or underscore with hyphen
	sanitized := regexp.MustCompile(`[^a-zA-Z0-9\-_]`).ReplaceAllString(value, "-")

	// Replace multiple consecutive hyphens with a single hyphen
	sanitized = regexp.MustCompile(`-+`).ReplaceAllString(sanitized, "-")

	// Trim leading and trailing hyphens
	sanitized = strings.Trim(sanitized, "-")

	// Convert to lowercase for consistency
	sanitized = strings.ToLower(sanitized)

	return sanitized
}
