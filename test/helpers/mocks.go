package helpers

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/ssm-access-manager/internal/models"
	"github.com/stretchr/testify/mock"
)

// MockApprovalGroupRepository is a mock implementation of ApprovalGroupRepository
type MockApprovalGroupRepository struct {
	mock.Mock
}

func (m *MockApprovalGroupRepository) SaveGroup(ctx context.Context, group *models.ApprovalGroup) error {
	args := m.Called(ctx, group)
	return args.Error(0)
}

func (m *MockApprovalGroupRepository) GetGroup(ctx context.Context, groupID string) (*models.ApprovalGroup, error) {
	args := m.Called(ctx, groupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ApprovalGroup), args.Error(1)
}

func (m *MockApprovalGroupRepository) ListAllGroups(ctx context.Context) ([]*models.ApprovalGroup, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ApprovalGroup), args.Error(1)
}

func (m *MockApprovalGroupRepository) ListGroupsByType(ctx context.Context, groupType models.ApprovalGroupType) ([]*models.ApprovalGroup, error) {
	args := m.Called(ctx, groupType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ApprovalGroup), args.Error(1)
}

func (m *MockApprovalGroupRepository) ListActiveManagerGroups(ctx context.Context) ([]*models.ApprovalGroup, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ApprovalGroup), args.Error(1)
}

func (m *MockApprovalGroupRepository) GetSecurityGroup(ctx context.Context) (*models.ApprovalGroup, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ApprovalGroup), args.Error(1)
}

func (m *MockApprovalGroupRepository) UpdateGroup(ctx context.Context, group *models.ApprovalGroup) error {
	args := m.Called(ctx, group)
	return args.Error(0)
}

func (m *MockApprovalGroupRepository) DeleteGroup(ctx context.Context, groupID string) error {
	args := m.Called(ctx, groupID)
	return args.Error(0)
}

// MockAuthorizationService is a mock implementation of AuthorizationService
type MockAuthorizationService struct {
	mock.Mock
}

func (m *MockAuthorizationService) VerifyAdministratorAuthorization(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockAuthorizationService) IsGroupMember(ctx context.Context, groupID, userID string) (bool, error) {
	args := m.Called(ctx, groupID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockAuthorizationService) LogUnauthorizedAttempt(ctx context.Context, userID, userName, action string) {
	m.Called(ctx, userID, userName, action)
}

// MockAuditService is a mock implementation of AuditService
type MockAuditService struct {
	mock.Mock
}

func (m *MockAuditService) LogApprovalGroupAdded(ctx context.Context, adminID, adminName string, group interface{}) {
	m.Called(ctx, adminID, adminName, group)
}

func (m *MockAuditService) LogApprovalGroupUpdated(ctx context.Context, adminID, adminName string, group interface{}) {
	m.Called(ctx, adminID, adminName, group)
}

func (m *MockAuditService) LogApprovalGroupRemoved(ctx context.Context, adminID, adminName, groupID string) {
	m.Called(ctx, adminID, adminName, groupID)
}

func (m *MockAuditService) LogRequestCreated(ctx context.Context, request *models.AccessRequest) {
	m.Called(ctx, request)
}

func (m *MockAuditService) LogSecurityApproval(ctx context.Context, approverID, approverName string, request *models.AccessRequest) {
	m.Called(ctx, approverID, approverName, request)
}

func (m *MockAuditService) LogManagerApproval(ctx context.Context, approverID, approverName string, request *models.AccessRequest) {
	m.Called(ctx, approverID, approverName, request)
}

func (m *MockAuditService) LogRequestDenied(ctx context.Context, denierID, denierName string, request *models.AccessRequest, reason string) {
	m.Called(ctx, denierID, denierName, request, reason)
}

func (m *MockAuditService) LogUnauthorizedApprovalAttempt(ctx context.Context, userID, userName, requestID string) {
	m.Called(ctx, userID, userName, requestID)
}

func (m *MockAuditService) LogSelfApprovalAttempt(ctx context.Context, userID, username, requestID string) {
	m.Called(ctx, userID, username, requestID)
}

func (m *MockAuditService) LogRequestRevoked(ctx context.Context, revokerID, revokerName string, request *models.AccessRequest, reason string) {
	m.Called(ctx, revokerID, revokerName, request, reason)
}

// MockGroupMembershipCache is a mock implementation of GroupMembershipCache
type MockGroupMembershipCache struct {
	mock.Mock
}

func (m *MockGroupMembershipCache) GetMembers(ctx context.Context, groupID string) ([]string, error) {
	args := m.Called(ctx, groupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockGroupMembershipCache) IsMember(ctx context.Context, groupID, userID string) (bool, error) {
	args := m.Called(ctx, groupID, userID)
	return args.Bool(0), args.Error(1)
}

// MockRequestRepository is a mock implementation of RequestRepository
type MockRequestRepository struct {
	mock.Mock
}

func (m *MockRequestRepository) SaveRequest(ctx context.Context, request *models.AccessRequest) error {
	args := m.Called(ctx, request)
	return args.Error(0)
}

func (m *MockRequestRepository) GetRequestByID(ctx context.Context, requestID string) (*models.AccessRequest, error) {
	args := m.Called(ctx, requestID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AccessRequest), args.Error(1)
}

func (m *MockRequestRepository) UpdateRequest(ctx context.Context, request *models.AccessRequest) error {
	args := m.Called(ctx, request)
	return args.Error(0)
}

func (m *MockRequestRepository) ListPendingRequests(ctx context.Context) ([]*models.AccessRequest, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AccessRequest), args.Error(1)
}

func (m *MockRequestRepository) DeleteRequest(ctx context.Context, requestID string) error {
	args := m.Called(ctx, requestID)
	return args.Error(0)
}

func (m *MockRequestRepository) ListRequestsByUser(ctx context.Context, userID string) ([]*models.AccessRequest, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AccessRequest), args.Error(1)
}

func (m *MockRequestRepository) ListRequestsByUsername(ctx context.Context, username string) ([]*models.AccessRequest, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AccessRequest), args.Error(1)
}

func (m *MockRequestRepository) ListAllRequests(ctx context.Context) ([]*models.AccessRequest, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AccessRequest), args.Error(1)
}

func (m *MockRequestRepository) GetRequest(ctx context.Context, requestID string) (*models.AccessRequest, error) {
	args := m.Called(ctx, requestID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AccessRequest), args.Error(1)
}

// MockRequestValidator is a mock implementation of RequestValidator
type MockRequestValidator struct {
	mock.Mock
}

func (m *MockRequestValidator) ValidateHost(host string) *models.ValidationResult {
	args := m.Called(host)
	return args.Get(0).(*models.ValidationResult)
}

func (m *MockRequestValidator) ValidatePort(port int) *models.ValidationResult {
	args := m.Called(port)
	return args.Get(0).(*models.ValidationResult)
}

func (m *MockRequestValidator) ValidateExpirationDate(expirationDate time.Time) *models.ValidationResult {
	args := m.Called(expirationDate)
	return args.Get(0).(*models.ValidationResult)
}

func (m *MockRequestValidator) ValidateUsername(username string) *models.ValidationResult {
	args := m.Called(username)
	return args.Get(0).(*models.ValidationResult)
}

func (m *MockRequestValidator) ValidateAccountID(accountID string) *models.ValidationResult {
	args := m.Called(accountID)
	return args.Get(0).(*models.ValidationResult)
}

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) SaveUser(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetUser(ctx context.Context, userID string) (*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) ListUsers(ctx context.Context) ([]*models.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepository) ListUsersByRole(ctx context.Context, role models.UserRole) ([]*models.User, error) {
	args := m.Called(ctx, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepository) UpdateUser(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateUserRole(ctx context.Context, userID string, role models.UserRole) error {
	args := m.Called(ctx, userID, role)
	return args.Error(0)
}

func (m *MockUserRepository) DeleteUser(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// MockDynamoDBClient is a mock implementation of DynamoDB client for testing
type MockDynamoDBClient struct {
	mock.Mock
}

func (m *MockDynamoDBClient) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dynamodb.PutItemOutput), args.Error(1)
}

func (m *MockDynamoDBClient) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dynamodb.GetItemOutput), args.Error(1)
}

func (m *MockDynamoDBClient) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dynamodb.QueryOutput), args.Error(1)
}

func (m *MockDynamoDBClient) Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dynamodb.ScanOutput), args.Error(1)
}

func (m *MockDynamoDBClient) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dynamodb.UpdateItemOutput), args.Error(1)
}

func (m *MockDynamoDBClient) DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dynamodb.DeleteItemOutput), args.Error(1)
}
