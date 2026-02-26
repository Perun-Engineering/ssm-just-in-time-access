package retry

import (
	"errors"
	"strings"
)

// RetryableError wraps an error to indicate it should be retried
type RetryableError struct {
	Err error
}

// Error implements the error interface
func (e *RetryableError) Error() string {
	return e.Err.Error()
}

// Unwrap implements the errors.Unwrap interface
func (e *RetryableError) Unwrap() error {
	return e.Err
}

// NewRetryableError creates a new retryable error
func NewRetryableError(err error) error {
	return &RetryableError{Err: err}
}

// NonRetryableError wraps an error to indicate it should not be retried
type NonRetryableError struct {
	Err error
}

// Error implements the error interface
func (e *NonRetryableError) Error() string {
	return e.Err.Error()
}

// Unwrap implements the errors.Unwrap interface
func (e *NonRetryableError) Unwrap() error {
	return e.Err
}

// NewNonRetryableError creates a new non-retryable error
func NewNonRetryableError(err error) error {
	return &NonRetryableError{Err: err}
}

// IsRetryable determines if an error should be retried
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check if explicitly marked as non-retryable
	var nonRetryable *NonRetryableError
	if errors.As(err, &nonRetryable) {
		return false
	}

	// Check if explicitly marked as retryable
	var retryable *RetryableError
	if errors.As(err, &retryable) {
		return true
	}

	// Check for common retryable error patterns
	errMsg := strings.ToLower(err.Error())

	// Network errors (typically retryable)
	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "connection reset") ||
		strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "temporary failure") ||
		strings.Contains(errMsg, "network is unreachable") {
		return true
	}

	// AWS throttling errors (retryable)
	if strings.Contains(errMsg, "throttling") ||
		strings.Contains(errMsg, "too many requests") ||
		strings.Contains(errMsg, "rate exceeded") ||
		strings.Contains(errMsg, "provisioned throughput exceeded") ||
		strings.Contains(errMsg, "request limit exceeded") {
		return true
	}

	// AWS service errors (retryable)
	if strings.Contains(errMsg, "service unavailable") ||
		strings.Contains(errMsg, "internal server error") ||
		strings.Contains(errMsg, "internal error") ||
		strings.Contains(errMsg, "503") ||
		strings.Contains(errMsg, "500") {
		return true
	}

	// DynamoDB specific errors (retryable)
	if strings.Contains(errMsg, "conditionalcheckfailed") {
		return true // Optimistic locking failure
	}

	// Slack API errors (retryable)
	if strings.Contains(errMsg, "rate_limited") ||
		strings.Contains(errMsg, "slack is down") {
		return true
	}

	// Default: non-retryable for unknown errors
	// This is conservative - better to fail fast than retry indefinitely
	return false
}

// IsNonRetryable determines if an error should not be retried
func IsNonRetryable(err error) bool {
	return !IsRetryable(err)
}

// Common error types for classification
var (
	// ErrValidation indicates a validation error (non-retryable)
	ErrValidation = errors.New("validation error")

	// ErrNotFound indicates a resource was not found (non-retryable)
	ErrNotFound = errors.New("not found")

	// ErrUnauthorized indicates an authorization error (non-retryable)
	ErrUnauthorized = errors.New("unauthorized")

	// ErrConflict indicates a conflict error (non-retryable)
	ErrConflict = errors.New("conflict")

	// ErrThrottled indicates a throttling error (retryable)
	ErrThrottled = errors.New("throttled")

	// ErrServiceUnavailable indicates a service unavailability (retryable)
	ErrServiceUnavailable = errors.New("service unavailable")

	// ErrTimeout indicates a timeout (retryable)
	ErrTimeout = errors.New("timeout")
)
