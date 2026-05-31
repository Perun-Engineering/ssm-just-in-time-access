package logging

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap logger with structured logging capabilities
type Logger struct {
	zap *zap.Logger
}

// NewLogger creates a new structured logger
func NewLogger(environment string) (*Logger, error) {
	var config zap.Config

	if environment == "production" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
	}

	// Use JSON encoding for structured logs
	config.Encoding = "json"

	// Set time format to ISO8601
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	zapLogger, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	return &Logger{zap: zapLogger}, nil
}

// NewDevelopmentLogger creates a logger for development
func NewDevelopmentLogger() (*Logger, error) {
	return NewLogger("development")
}

// NewProductionLogger creates a logger for production
func NewProductionLogger() (*Logger, error) {
	return NewLogger("production")
}

// WithFields returns a logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return &Logger{zap: l.zap.With(zapFields...)}
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.zap.Info(msg, fields...)
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.zap.Error(msg, fields...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.zap.Warn(msg, fields...)
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.zap.Debug(msg, fields...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.zap.Fatal(msg, fields...)
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.zap.Sync()
}

// LogAccessRequest logs an access request event
func (l *Logger) LogAccessRequest(ctx context.Context, requestID, username, userID, host string, port int, accountID string, expirationDate time.Time) {
	l.Info("access_request_created",
		zap.String("event_type", "access_request"),
		zap.String("request_id", requestID),
		zap.String("username", username),
		zap.String("user_id", userID),
		zap.String("host", host),
		zap.Int("port", port),
		zap.String("account_id", accountID),
		zap.Time("expiration_date", expirationDate),
	)
}

// LogApprovalDecision logs an approval or denial decision
func (l *Logger) LogApprovalDecision(ctx context.Context, requestID, approver, approverID, decision string) {
	l.Info("approval_decision",
		zap.String("event_type", "approval"),
		zap.String("request_id", requestID),
		zap.String("approver", approver),
		zap.String("approver_id", approverID),
		zap.String("decision", decision),
	)
}

// LogDocumentCreation logs an SSM document creation event
func (l *Logger) LogDocumentCreation(ctx context.Context, documentID, documentName, accountID, username, host string, port int) {
	l.Info("document_created",
		zap.String("event_type", "document_lifecycle"),
		zap.String("action", "create"),
		zap.String("document_id", documentID),
		zap.String("document_name", documentName),
		zap.String("account_id", accountID),
		zap.String("username", username),
		zap.String("host", host),
		zap.Int("port", port),
	)
}

// LogDocumentDeletion logs an SSM document deletion event
func (l *Logger) LogDocumentDeletion(ctx context.Context, documentID, documentName, accountID, reason string) {
	l.Info("document_deleted",
		zap.String("event_type", "document_lifecycle"),
		zap.String("action", "delete"),
		zap.String("document_id", documentID),
		zap.String("document_name", documentName),
		zap.String("account_id", accountID),
		zap.String("reason", reason),
	)
}

// LogRoleAssumption logs a role assumption attempt
func (l *Logger) LogRoleAssumption(ctx context.Context, accountID, roleName, region string, success bool, err error) {
	fields := []zap.Field{
		zap.String("event_type", "role_assumption"),
		zap.String("account_id", accountID),
		zap.String("role_name", roleName),
		zap.String("region", region),
		zap.Bool("success", success),
	}

	if err != nil {
		fields = append(fields, zap.Error(err))
	}

	if success {
		l.Info("role_assumption_success", fields...)
	} else {
		l.Error("role_assumption_failed", fields...)
	}
}

// LogUnauthorizedAttempt logs an unauthorized access attempt
func (l *Logger) LogUnauthorizedAttempt(ctx context.Context, userID, action string) {
	l.Warn("unauthorized_attempt",
		zap.String("event_type", "security"),
		zap.String("user_id", userID),
		zap.String("action", action),
	)
}

// LogError logs a general error with context
func (l *Logger) LogError(ctx context.Context, operation string, err error, fields map[string]interface{}) {
	zapFields := []zap.Field{
		zap.String("operation", operation),
		zap.Error(err),
	}

	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}

	l.Error("operation_failed", zapFields...)
}

// SanitizeForLogging removes sensitive information from a value
func SanitizeForLogging(value string) string {
	// Don't log credentials, secrets, or tokens
	if len(value) > 20 {
		return value[:10] + "..." + value[len(value)-4:]
	}
	return "***"
}
