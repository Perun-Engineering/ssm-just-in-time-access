package helpers

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/ssm-access-manager/internal/models"
)

// ApprovalGroupRepositoryInterface defines the interface for approval group repository
type ApprovalGroupRepositoryInterface interface {
	SaveGroup(ctx context.Context, group *models.ApprovalGroup) error
	GetGroup(ctx context.Context, groupID string) (*models.ApprovalGroup, error)
	ListAllGroups(ctx context.Context) ([]*models.ApprovalGroup, error)
	ListGroupsByType(ctx context.Context, groupType models.ApprovalGroupType) ([]*models.ApprovalGroup, error)
	ListActiveManagerGroups(ctx context.Context) ([]*models.ApprovalGroup, error)
	GetSecurityGroup(ctx context.Context) (*models.ApprovalGroup, error)
	UpdateGroup(ctx context.Context, group *models.ApprovalGroup) error
	DeleteGroup(ctx context.Context, groupID string) error
}

// AuthorizationServiceInterface defines the interface for authorization service
type AuthorizationServiceInterface interface {
	VerifyAdministratorAuthorization(ctx context.Context, userID string) error
	IsGroupMember(ctx context.Context, groupID, userID string) (bool, error)
	LogUnauthorizedAttempt(ctx context.Context, userID, userName, action string)
}

// AuditServiceInterface defines the interface for audit service
type AuditServiceInterface interface {
	LogApprovalGroupAdded(ctx context.Context, adminID, adminName string, group interface{})
	LogApprovalGroupUpdated(ctx context.Context, adminID, adminName string, group interface{})
	LogApprovalGroupRemoved(ctx context.Context, adminID, adminName, groupID string)
	LogRequestCreated(ctx context.Context, request *models.AccessRequest)
	LogSecurityApproval(ctx context.Context, approverID, approverName string, request *models.AccessRequest)
	LogManagerApproval(ctx context.Context, approverID, approverName string, request *models.AccessRequest)
	LogRequestDenied(ctx context.Context, denierID, denierName string, request *models.AccessRequest, reason string)
	LogUnauthorizedApprovalAttempt(ctx context.Context, userID, userName, requestID string)
	LogSelfApprovalAttempt(ctx context.Context, userID, username, requestID string)
}

// DynamoDBClientInterface defines the interface for DynamoDB operations
type DynamoDBClientInterface interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
}
