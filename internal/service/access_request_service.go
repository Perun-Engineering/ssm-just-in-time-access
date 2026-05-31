package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ssm-access-manager/internal/models"
	"github.com/ssm-access-manager/internal/repository"
)

// AccessRequestService handles access request business logic
type AccessRequestService struct {
	requestRepo       repository.RequestRepositoryInterface
	validator         RequestValidatorInterface
	authService       AuthorizationServiceInterface
	auditService      AuditServiceInterface
	documentRepo      *repository.DocumentRepository
	documentService   *SSMDocumentService
	allowSelfApproval bool // For testing purposes only - defaults to false
	slackNotifier     interface {
		SendRevocationNotification(ctx context.Context, userID string, request *models.AccessRequest, revokedBy, reason string) error
	}
}

// NewAccessRequestService creates a new access request service
func NewAccessRequestService(
	requestRepo repository.RequestRepositoryInterface,
	validator RequestValidatorInterface,
	authService AuthorizationServiceInterface,
	auditService AuditServiceInterface,
) *AccessRequestService {
	return &AccessRequestService{
		requestRepo:       requestRepo,
		validator:         validator,
		authService:       authService,
		auditService:      auditService,
		documentRepo:      nil,   // Set via SetDocumentRepository if needed
		documentService:   nil,   // Set via SetDocumentService if needed
		slackNotifier:     nil,   // Set via SetSlackNotifier if needed
		allowSelfApproval: false, // Default to false for security
	}
}

// SetDocumentRepository sets the document repository (for revoke functionality)
func (s *AccessRequestService) SetDocumentRepository(repo *repository.DocumentRepository) {
	s.documentRepo = repo
}

// SetDocumentService sets the document service (for revoke functionality)
func (s *AccessRequestService) SetDocumentService(service *SSMDocumentService) {
	s.documentService = service
}

// SetSlackNotifier sets the Slack notifier (for revoke functionality)
func (s *AccessRequestService) SetSlackNotifier(notifier interface {
	SendRevocationNotification(ctx context.Context, userID string, request *models.AccessRequest, revokedBy, reason string) error
}) {
	s.slackNotifier = notifier
}

// SetAllowSelfApproval sets whether self-approval is allowed (for testing purposes only)
// WARNING: This should only be enabled in test/development environments
func (s *AccessRequestService) SetAllowSelfApproval(allow bool) {
	s.allowSelfApproval = allow
}

// CreateRequest creates a new access request
// CreateRequest creates a new access request
func (s *AccessRequestService) CreateRequest(
	ctx context.Context,
	username, userID, host string,
	port int,
	accountID string,
	expirationDate time.Time,
	managerGroupID, managerGroupName string,
	reason string,
) (*models.AccessRequest, error) {
	// Validate inputs
	if result := s.validator.ValidateHost(host); !result.IsValid {
		return nil, fmt.Errorf("invalid host: %s", result.ErrorMessage)
	}

	if result := s.validator.ValidatePort(port); !result.IsValid {
		return nil, fmt.Errorf("invalid port: %s", result.ErrorMessage)
	}

	if result := s.validator.ValidateExpirationDate(expirationDate); !result.IsValid {
		return nil, fmt.Errorf("invalid expiration date: %s", result.ErrorMessage)
	}

	if result := s.validator.ValidateUsername(username); !result.IsValid {
		return nil, fmt.Errorf("invalid username: %s", result.ErrorMessage)
	}

	if result := s.validator.ValidateAccountID(accountID); !result.IsValid {
		return nil, fmt.Errorf("invalid account ID: %s", result.ErrorMessage)
	}

	// Validate manager group selection (required for all requests)
	if managerGroupID == "" || managerGroupName == "" {
		return nil, fmt.Errorf("manager group selection is required")
	}

	// Create the request
	now := time.Now()
	request := &models.AccessRequest{
		RequestID:        uuid.New().String(),
		Username:         username,
		UserID:           userID,
		Host:             host,
		Port:             port,
		AccountID:        accountID,
		ExpirationDate:   expirationDate,
		ManagerGroupID:   managerGroupID,
		ManagerGroupName: managerGroupName,
		Reason:           reason,
		Status:           models.RequestStatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Save to repository
	err := s.requestRepo.SaveRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to save request: %w", err)
	}

	// Log request creation
	if s.auditService != nil {
		s.auditService.LogRequestCreated(ctx, request)
	}

	return request, nil
}

// ApproveRequest approves an access request
func (s *AccessRequestService) ApproveRequest(
	ctx context.Context,
	requestID, approverID, approverName string,
) (*models.AccessRequest, error) {
	// Note: Authorization check should be done by caller (checking group membership)
	// This method assumes the caller has already verified authorization

	// Get the request
	request, err := s.requestRepo.GetRequestByID(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get request: %w", err)
	}

	// Check if request is pending
	if !request.IsPending() {
		return nil, fmt.Errorf("request is not pending (current status: %s)", request.Status)
	}

	// Update request status
	now := time.Now()
	request.Status = models.RequestStatusApproved
	request.Approver = &approverName
	request.ApproverID = &approverID
	request.ApprovalTimestamp = &now
	request.UpdatedAt = now

	// Save updated request
	err = s.requestRepo.SaveRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to update request: %w", err)
	}

	return request, nil
}

// DenyRequest denies an access request
// DenyRequest denies an access request
func (s *AccessRequestService) DenyRequest(
	ctx context.Context,
	requestID, approverID, approverName, reason string,
) (*models.AccessRequest, error) {
	// Note: Authorization check should be done by caller (checking group membership)
	// This method assumes the caller has already verified authorization

	// Get the request
	request, err := s.requestRepo.GetRequestByID(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get request: %w", err)
	}

	// Check if request is pending or partially approved
	if !request.IsPending() && !request.IsPartiallyApproved() {
		return nil, fmt.Errorf("request cannot be denied (current status: %s)", request.Status)
	}

	// Update request status
	now := time.Now()
	request.Status = models.RequestStatusDenied
	request.Approver = &approverName
	request.ApproverID = &approverID
	request.ApprovalTimestamp = &now
	request.DenialReason = &reason
	request.UpdatedAt = now

	// Save updated request
	err = s.requestRepo.SaveRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to update request: %w", err)
	}

	// Log denial
	if s.auditService != nil {
		s.auditService.LogRequestDenied(ctx, approverID, approverName, request, reason)
	}

	return request, nil
}

// GetRequest retrieves an access request by ID
func (s *AccessRequestService) GetRequest(ctx context.Context, requestID string) (*models.AccessRequest, error) {
	request, err := s.requestRepo.GetRequestByID(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get request: %w", err)
	}
	return request, nil
}

// ListUserRequests lists all requests for a specific user
func (s *AccessRequestService) ListUserRequests(ctx context.Context, username string) ([]*models.AccessRequest, error) {
	requests, err := s.requestRepo.ListRequestsByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to list user requests: %w", err)
	}
	return requests, nil
}

// ListPendingRequests lists all pending requests
func (s *AccessRequestService) ListPendingRequests(ctx context.Context) ([]*models.AccessRequest, error) {
	requests, err := s.requestRepo.ListPendingRequests(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending requests: %w", err)
	}
	return requests, nil
}

// ListAllRequests lists all requests regardless of status
func (s *AccessRequestService) ListAllRequests(ctx context.Context) ([]*models.AccessRequest, error) {
	requests, err := s.requestRepo.ListAllRequests(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list all requests: %w", err)
	}
	return requests, nil
}

// ValidateRequestParameters validates request parameters and returns missing fields
func (s *AccessRequestService) ValidateRequestParameters(
	host string,
	port int,
	accountID string,
	expirationDate time.Time,
) (bool, []string) {
	var missingFields []string

	if host == "" {
		missingFields = append(missingFields, "host")
	} else if result := s.validator.ValidateHost(host); !result.IsValid {
		missingFields = append(missingFields, fmt.Sprintf("host (%s)", result.ErrorMessage))
	}

	if port == 0 {
		missingFields = append(missingFields, "port")
	} else if result := s.validator.ValidatePort(port); !result.IsValid {
		missingFields = append(missingFields, fmt.Sprintf("port (%s)", result.ErrorMessage))
	}

	if accountID == "" {
		missingFields = append(missingFields, "account")
	} else if result := s.validator.ValidateAccountID(accountID); !result.IsValid {
		missingFields = append(missingFields, fmt.Sprintf("account (%s)", result.ErrorMessage))
	}

	if expirationDate.IsZero() {
		missingFields = append(missingFields, "expires")
	} else if result := s.validator.ValidateExpirationDate(expirationDate); !result.IsValid {
		missingFields = append(missingFields, fmt.Sprintf("expires (%s)", result.ErrorMessage))
	}

	return len(missingFields) == 0, missingFields
}

// RevokeRequest revokes an approved access request and deletes the associated SSM document
// RevokeRequest revokes an approved access request and deletes the associated SSM document
func (s *AccessRequestService) RevokeRequest(
	ctx context.Context,
	requestID, revokerID, revokerName, reason string,
) (*models.AccessRequest, error) {
	// Verify administrator authorization
	err := s.authService.VerifyAdministratorAuthorization(ctx, revokerID)
	if err != nil {
		s.authService.LogUnauthorizedAttempt(ctx, revokerID, revokerName, requestID)
		return nil, fmt.Errorf("unauthorized: only administrators can revoke requests: %w", err)
	}

	// Get the request
	request, err := s.requestRepo.GetRequestByID(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get request: %w", err)
	}

	// Check if request is approved
	if !request.IsApproved() {
		return nil, fmt.Errorf("request is not approved (current status: %s)", request.Status)
	}

	// Find and delete the SSM document if document repository and service are available
	if s.documentRepo != nil && s.documentService != nil {
		document, err := s.documentRepo.GetDocumentByRequestID(ctx, requestID)
		if err != nil {
			// Log but don't fail - document might not exist yet or already deleted
			fmt.Printf("Warning: Document not found for request %s: %v\n", requestID, err)
		} else {
			// Delete the SSM document from AWS
			err = s.documentService.DeleteDocument(ctx, document)
			if err != nil {
				return nil, fmt.Errorf("failed to delete SSM document: %w", err)
			}

			// Delete document from DynamoDB
			err = s.documentRepo.DeleteDocument(ctx, document.DocumentID)
			if err != nil {
				fmt.Printf("Warning: Failed to delete document from DB: %v\n", err)
			}
		}
	}

	// Update request status
	now := time.Now()
	request.Status = models.RequestStatusRevoked
	request.RevokedBy = &revokerName
	request.RevokedByID = &revokerID
	request.RevokedAt = &now
	request.RevocationReason = &reason
	request.UpdatedAt = now

	// Save updated request
	err = s.requestRepo.SaveRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to update request: %w", err)
	}

	// Log revocation
	if s.auditService != nil {
		s.auditService.LogRequestRevoked(ctx, revokerID, revokerName, request, reason)
	}

	// Send notification to user if notifier is available
	if s.slackNotifier != nil {
		err = s.slackNotifier.SendRevocationNotification(ctx, request.UserID, request, revokerName, reason)
		if err != nil {
			fmt.Printf("Warning: Failed to send revocation notification: %v\n", err)
		}
	}

	return request, nil
}

// ApproveRequestSecurity grants security approval for an access request
func (s *AccessRequestService) ApproveRequestSecurity(
	ctx context.Context,
	requestID, approverID, approverName string,
) (*models.AccessRequest, error) {
	// Get the request
	request, err := s.requestRepo.GetRequestByID(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get request: %w", err)
	}

	// Check for self-approval attempt (unless explicitly allowed for testing)
	if !s.allowSelfApproval && request.UserID == approverID {
		// Log self-approval attempt to audit trail
		if s.auditService != nil {
			s.auditService.LogSelfApprovalAttempt(ctx, approverID, approverName, requestID)
		}
		return nil, fmt.Errorf("you cannot approve your own access request")
	}

	// Check if request is pending or partially approved
	if !request.IsPending() && !request.IsPartiallyApproved() {
		return nil, fmt.Errorf("request cannot be approved (current status: %s)", request.Status)
	}

	// Check if security approval already granted
	if request.HasSecurityApproval() {
		return nil, fmt.Errorf("security approval already granted")
	}

	// Update request with security approval
	now := time.Now()
	request.SecurityApproverID = &approverID
	request.SecurityApproverName = &approverName
	request.SecurityApprovalTimestamp = &now
	request.UpdatedAt = now

	// Check if both approvals are now present
	if request.HasManagerApproval() {
		// Both approvals granted - mark as fully approved
		request.Status = models.RequestStatusApproved
		// Set legacy fields for backward compatibility
		request.Approver = &approverName
		request.ApproverID = &approverID
		request.ApprovalTimestamp = &now
	} else {
		// Only security approval granted - mark as partially approved
		request.Status = models.RequestStatusPartiallyApproved
	}

	// Save updated request
	err = s.requestRepo.SaveRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to update request: %w", err)
	}

	// Log security approval
	if s.auditService != nil {
		s.auditService.LogSecurityApproval(ctx, approverID, approverName, request)
	}

	return request, nil
}

// ApproveRequestManager grants manager approval for an access request
func (s *AccessRequestService) ApproveRequestManager(
	ctx context.Context,
	requestID, approverID, approverName string,
) (*models.AccessRequest, error) {
	// Get the request
	request, err := s.requestRepo.GetRequestByID(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get request: %w", err)
	}

	// Check for self-approval attempt (unless explicitly allowed for testing)
	if !s.allowSelfApproval && request.UserID == approverID {
		// Log self-approval attempt to audit trail
		if s.auditService != nil {
			s.auditService.LogSelfApprovalAttempt(ctx, approverID, approverName, requestID)
		}
		return nil, fmt.Errorf("you cannot approve your own access request")
	}

	// Check if request is pending or partially approved
	if !request.IsPending() && !request.IsPartiallyApproved() {
		return nil, fmt.Errorf("request cannot be approved (current status: %s)", request.Status)
	}

	// Check if manager approval already granted
	if request.HasManagerApproval() {
		return nil, fmt.Errorf("manager approval already granted")
	}

	// Update request with manager approval
	now := time.Now()
	request.ManagerApproverID = &approverID
	request.ManagerApproverName = &approverName
	request.ManagerApprovalTimestamp = &now
	request.UpdatedAt = now

	// Check if both approvals are now present
	if request.HasSecurityApproval() {
		// Both approvals granted - mark as fully approved
		request.Status = models.RequestStatusApproved
		// Set legacy fields for backward compatibility
		request.Approver = &approverName
		request.ApproverID = &approverID
		request.ApprovalTimestamp = &now
	} else {
		// Only manager approval granted - mark as partially approved
		request.Status = models.RequestStatusPartiallyApproved
	}

	// Save updated request
	err = s.requestRepo.SaveRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to update request: %w", err)
	}

	// Log manager approval
	if s.auditService != nil {
		s.auditService.LogManagerApproval(ctx, approverID, approverName, request)
	}

	return request, nil
}
