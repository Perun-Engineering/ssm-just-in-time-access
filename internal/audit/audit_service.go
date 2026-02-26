package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ssm-access-manager/internal/logging"
	"github.com/ssm-access-manager/internal/models"
)

// AuditLogService handles immutable audit logging to CloudWatch
type AuditLogService struct {
	logger   *logging.Logger
	logGroup string
	teamID   string
}

// AuditLogEntry represents a single audit log entry
type AuditLogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	EventType string                 `json:"event_type"`
	EventID   string                 `json:"event_id"`
	Actor     AuditActor             `json:"actor"`
	Target    AuditTarget            `json:"target"`
	Details   map[string]interface{} `json:"details"`
	Metadata  AuditMetadata          `json:"metadata"`
}

// AuditActor represents the user who performed the action
type AuditActor struct {
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	SlackTeamID string `json:"slack_team_id"`
}

// AuditTarget represents the resource being acted upon
type AuditTarget struct {
	RequestID    string `json:"request_id,omitempty"`
	ResourceType string `json:"resource_type"`
}

// AuditMetadata contains additional context about the action
type AuditMetadata struct {
	SourceIP        string `json:"source_ip,omitempty"`
	UserAgent       string `json:"user_agent,omitempty"`
	LambdaRequestID string `json:"lambda_request_id"`
}

// Event type constants
const (
	EventRequestCreated              = "request_created"
	EventRequestApprovedSecurity     = "request_approved_security"
	EventRequestApprovedManager      = "request_approved_manager"
	EventRequestFullyApproved        = "request_fully_approved"
	EventRequestDenied               = "request_denied"
	EventRequestRevoked              = "request_revoked"
	EventSelfApprovalAttempt         = "self_approval_attempt"
	EventDocumentCreated             = "document_created"
	EventDocumentDeleted             = "document_deleted"

	EventApprovalGroupAdded          = "approval_group_added"
	EventApprovalGroupUpdated        = "approval_group_updated"
	EventApprovalGroupRemoved        = "approval_group_removed"
	EventUnauthorizedApprovalAttempt = "unauthorized_approval_attempt"
	EventAccountAdded                = "account_added"
	EventAccountUpdated              = "account_updated"
	EventAccountRemoved              = "account_removed"
)

// NewAuditLogService creates a new audit log service
func NewAuditLogService(logger *logging.Logger, logGroup, teamID string) *AuditLogService {
	return &AuditLogService{
		logger:   logger,
		logGroup: logGroup,
		teamID:   teamID,
	}
}

// writeAuditLog writes an audit log entry to CloudWatch Logs as structured JSON
func (s *AuditLogService) writeAuditLog(entry AuditLogEntry) {
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to marshal audit log: %v", err))
		return
	}

	// Write to CloudWatch Logs via structured logging
	s.logger.Info(string(jsonBytes))
}

// getMetadata extracts metadata from context
func (s *AuditLogService) getMetadata(ctx context.Context) AuditMetadata {
	metadata := AuditMetadata{
		LambdaRequestID: getLambdaRequestID(ctx),
		SourceIP:        getSourceIP(ctx),
		UserAgent:       getUserAgent(ctx),
	}
	return metadata
}

// Helper functions to extract context values
func getLambdaRequestID(ctx context.Context) string {
	if requestID := ctx.Value("lambda_request_id"); requestID != nil {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}

func getSourceIP(ctx context.Context) string {
	if sourceIP := ctx.Value("source_ip"); sourceIP != nil {
		if ip, ok := sourceIP.(string); ok {
			return ip
		}
	}
	return ""
}

func getUserAgent(ctx context.Context) string {
	if userAgent := ctx.Value("user_agent"); userAgent != nil {
		if ua, ok := userAgent.(string); ok {
			return ua
		}
	}
	return ""
}


// LogRequestCreated logs when a new access request is created
func (s *AuditLogService) LogRequestCreated(ctx context.Context, request *models.AccessRequest) {
	entry := AuditLogEntry{
		Timestamp: time.Now(),
		EventType: EventRequestCreated,
		EventID:   uuid.New().String(),
		Actor: AuditActor{
			UserID:      request.UserID,
			Username:    request.Username,
			SlackTeamID: s.teamID,
		},
		Target: AuditTarget{
			RequestID:    request.RequestID,
			ResourceType: "access_request",
		},
		Details: map[string]interface{}{
			"host":               request.Host,
			"port":               request.Port,
			"account_id":         request.AccountID,
			"expiration_date":    request.ExpirationDate.Format(time.RFC3339),
			"manager_group_id":   request.ManagerGroupID,
			"manager_group_name": request.ManagerGroupName,
			"reason":             request.Reason,
		},
		Metadata: s.getMetadata(ctx),
	}

	s.writeAuditLog(entry)
}

// LogSecurityApproval logs when security approval is granted
func (s *AuditLogService) LogSecurityApproval(ctx context.Context, approverID, approverName string, request *models.AccessRequest) {
	entry := AuditLogEntry{
		Timestamp: time.Now(),
		EventType: EventRequestApprovedSecurity,
		EventID:   uuid.New().String(),
		Actor: AuditActor{
			UserID:      approverID,
			Username:    approverName,
			SlackTeamID: s.teamID,
		},
		Target: AuditTarget{
			RequestID:    request.RequestID,
			ResourceType: "access_request",
		},
		Details: map[string]interface{}{
			"host":       request.Host,
			"port":       request.Port,
			"account_id": request.AccountID,
			"requester":  request.Username,
		},
		Metadata: s.getMetadata(ctx),
	}

	s.writeAuditLog(entry)
}

// LogManagerApproval logs when manager approval is granted
func (s *AuditLogService) LogManagerApproval(ctx context.Context, approverID, approverName string, request *models.AccessRequest) {
	entry := AuditLogEntry{
		Timestamp: time.Now(),
		EventType: EventRequestApprovedManager,
		EventID:   uuid.New().String(),
		Actor: AuditActor{
			UserID:      approverID,
			Username:    approverName,
			SlackTeamID: s.teamID,
		},
		Target: AuditTarget{
			RequestID:    request.RequestID,
			ResourceType: "access_request",
		},
		Details: map[string]interface{}{
			"host":              request.Host,
			"port":              request.Port,
			"account_id":        request.AccountID,
			"requester":         request.Username,
			"manager_group_id":  request.ManagerGroupID,
			"manager_group_name": request.ManagerGroupName,
		},
		Metadata: s.getMetadata(ctx),
	}

	s.writeAuditLog(entry)
}

// LogRequestDenied logs when a request is denied
func (s *AuditLogService) LogRequestDenied(ctx context.Context, denierID, denierName string, request *models.AccessRequest, reason string) {
	entry := AuditLogEntry{
		Timestamp: time.Now(),
		EventType: EventRequestDenied,
		EventID:   uuid.New().String(),
		Actor: AuditActor{
			UserID:      denierID,
			Username:    denierName,
			SlackTeamID: s.teamID,
		},
		Target: AuditTarget{
			RequestID:    request.RequestID,
			ResourceType: "access_request",
		},
		Details: map[string]interface{}{
			"host":       request.Host,
			"port":       request.Port,
			"account_id": request.AccountID,
			"requester":  request.Username,
			"reason":     reason,
		},
		Metadata: s.getMetadata(ctx),
	}

	s.writeAuditLog(entry)
}

// LogSelfApprovalAttempt logs when a user attempts to approve their own request
func (s *AuditLogService) LogSelfApprovalAttempt(ctx context.Context, userID, username, requestID string) {
	entry := AuditLogEntry{
		Timestamp: time.Now(),
		EventType: EventSelfApprovalAttempt,
		EventID:   uuid.New().String(),
		Actor: AuditActor{
			UserID:      userID,
			Username:    username,
			SlackTeamID: s.teamID,
		},
		Target: AuditTarget{
			RequestID:    requestID,
			ResourceType: "access_request",
		},
		Details: map[string]interface{}{
			"blocked": true,
			"reason":  "User attempted to approve their own request",
		},
		Metadata: s.getMetadata(ctx),
	}

	s.writeAuditLog(entry)
}

// LogRequestRevoked logs when a request is revoked
func (s *AuditLogService) LogRequestRevoked(ctx context.Context, revokerID, revokerName string, request *models.AccessRequest, reason string) {
	entry := AuditLogEntry{
		Timestamp: time.Now(),
		EventType: EventRequestRevoked,
		EventID:   uuid.New().String(),
		Actor: AuditActor{
			UserID:      revokerID,
			Username:    revokerName,
			SlackTeamID: s.teamID,
		},
		Target: AuditTarget{
			RequestID:    request.RequestID,
			ResourceType: "access_request",
		},
		Details: map[string]interface{}{
			"host":       request.Host,
			"port":       request.Port,
			"account_id": request.AccountID,
			"requester":  request.Username,
			"reason":     reason,
		},
		Metadata: s.getMetadata(ctx),
	}

	s.writeAuditLog(entry)
}

// LogDocumentCreated logs when an SSM document is created
func (s *AuditLogService) LogDocumentCreated(ctx context.Context, document *models.SSMDocument) {
	entry := AuditLogEntry{
		Timestamp: time.Now(),
		EventType: EventDocumentCreated,
		EventID:   uuid.New().String(),
		Actor: AuditActor{
			UserID:      "system",
			Username:    "system",
			SlackTeamID: s.teamID,
		},
		Target: AuditTarget{
			RequestID:    document.RequestID,
			ResourceType: "ssm_document",
		},
		Details: map[string]interface{}{
			"document_id":   document.DocumentID,
			"document_name": document.DocumentName,
			"account_id":    document.AccountID,
			"region":        document.Region,
			"host":          document.Host,
			"port":          document.Port,
			"expires_at":    document.ExpiresAt.Format(time.RFC3339),
		},
		Metadata: s.getMetadata(ctx),
	}

	s.writeAuditLog(entry)
}

// LogDocumentDeleted logs when an SSM document is deleted
func (s *AuditLogService) LogDocumentDeleted(ctx context.Context, document *models.SSMDocument, reason string) {
	entry := AuditLogEntry{
		Timestamp: time.Now(),
		EventType: EventDocumentDeleted,
		EventID:   uuid.New().String(),
		Actor: AuditActor{
			UserID:      "system",
			Username:    "system",
			SlackTeamID: s.teamID,
		},
		Target: AuditTarget{
			RequestID:    document.RequestID,
			ResourceType: "ssm_document",
		},
		Details: map[string]interface{}{
			"document_id":   document.DocumentID,
			"document_name": document.DocumentName,
			"account_id":    document.AccountID,
			"region":        document.Region,
			"reason":        reason,
		},
		Metadata: s.getMetadata(ctx),
	}

	s.writeAuditLog(entry)
}

// LogUnauthorizedApprovalAttempt logs when someone tries to approve without permission
func (s *AuditLogService) LogUnauthorizedApprovalAttempt(ctx context.Context, userID, userName, requestID string) {
	entry := AuditLogEntry{
		Timestamp: time.Now(),
		EventType: EventUnauthorizedApprovalAttempt,
		EventID:   uuid.New().String(),
		Actor: AuditActor{
			UserID:      userID,
			Username:    userName,
			SlackTeamID: s.teamID,
		},
		Target: AuditTarget{
			RequestID:    requestID,
			ResourceType: "access_request",
		},
		Details: map[string]interface{}{
			"action": "approve_request",
		},
		Metadata: s.getMetadata(ctx),
	}

	s.writeAuditLog(entry)
}

// LogApprovalGroupAdded logs when an approval group is added
func (s *AuditLogService) LogApprovalGroupAdded(ctx context.Context, adminID, adminName string, group interface{}) {
	// Note: group parameter is interface{} to avoid circular dependency
	// It should be *models.ApprovalGroup but we'll handle it generically
	entry := AuditLogEntry{
		Timestamp: time.Now(),
		EventType: EventApprovalGroupAdded,
		EventID:   uuid.New().String(),
		Actor: AuditActor{
			UserID:      adminID,
			Username:    adminName,
			SlackTeamID: s.teamID,
		},
		Target: AuditTarget{
			ResourceType: "approval_group",
		},
		Details: map[string]interface{}{
			"group": group,
		},
		Metadata: s.getMetadata(ctx),
	}

	s.writeAuditLog(entry)
}

// LogApprovalGroupUpdated logs when an approval group is updated
func (s *AuditLogService) LogApprovalGroupUpdated(ctx context.Context, adminID, adminName string, group interface{}) {
	entry := AuditLogEntry{
		Timestamp: time.Now(),
		EventType: EventApprovalGroupUpdated,
		EventID:   uuid.New().String(),
		Actor: AuditActor{
			UserID:      adminID,
			Username:    adminName,
			SlackTeamID: s.teamID,
		},
		Target: AuditTarget{
			ResourceType: "approval_group",
		},
		Details: map[string]interface{}{
			"group": group,
		},
		Metadata: s.getMetadata(ctx),
	}

	s.writeAuditLog(entry)
}

// LogApprovalGroupRemoved logs when an approval group is removed
func (s *AuditLogService) LogApprovalGroupRemoved(ctx context.Context, adminID, adminName, groupID string) {
	entry := AuditLogEntry{
		Timestamp: time.Now(),
		EventType: EventApprovalGroupRemoved,
		EventID:   uuid.New().String(),
		Actor: AuditActor{
			UserID:      adminID,
			Username:    adminName,
			SlackTeamID: s.teamID,
		},
		Target: AuditTarget{
			ResourceType: "approval_group",
		},
		Details: map[string]interface{}{
			"group_id": groupID,
		},
		Metadata: s.getMetadata(ctx),
	}

	s.writeAuditLog(entry)
}
