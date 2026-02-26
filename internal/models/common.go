package models

// ValidationResult represents the result of a validation operation
type ValidationResult struct {
	IsValid      bool
	ErrorMessage string
}

// NewValidationResult creates a new validation result
func NewValidationResult(isValid bool, errorMessage string) ValidationResult {
	return ValidationResult{
		IsValid:      isValid,
		ErrorMessage: errorMessage,
	}
}

// Valid creates a valid validation result
func Valid() ValidationResult {
	return ValidationResult{IsValid: true}
}

// Invalid creates an invalid validation result with an error message
func Invalid(errorMessage string) ValidationResult {
	return ValidationResult{
		IsValid:      false,
		ErrorMessage: errorMessage,
	}
}

// ApprovalDecision represents an approval or denial decision
type ApprovalDecision struct {
	Approved bool
	Reason   string
}

// NewApprovalDecision creates a new approval decision
func NewApprovalDecision(approved bool, reason string) ApprovalDecision {
	return ApprovalDecision{
		Approved: approved,
		Reason:   reason,
	}
}

// Approve creates an approval decision
func Approve() ApprovalDecision {
	return ApprovalDecision{Approved: true}
}

// Deny creates a denial decision with a reason
func Deny(reason string) ApprovalDecision {
	return ApprovalDecision{
		Approved: false,
		Reason:   reason,
	}
}

// ExpirationReport represents a report of expiration processing
type ExpirationReport struct {
	TotalProcessed      int
	SuccessfulDeletions int
	FailedDeletions     int
	Errors              []string
}

// NewExpirationReport creates a new expiration report
func NewExpirationReport() *ExpirationReport {
	return &ExpirationReport{
		Errors: make([]string, 0),
	}
}

// AddError adds an error to the report
func (r *ExpirationReport) AddError(err string) {
	r.Errors = append(r.Errors, err)
}

// HasErrors returns true if there are any errors
func (r *ExpirationReport) HasErrors() bool {
	return len(r.Errors) > 0
}
