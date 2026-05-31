package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/ssm-access-manager/internal/models"
	"github.com/ssm-access-manager/internal/repository"
	"github.com/ssm-access-manager/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRequestRepository_SaveRequest(t *testing.T) {
	tests := []struct {
		name        string
		request     *models.AccessRequest
		setupMock   func(*helpers.MockDynamoDBClient)
		expectError bool
		errorMsg    string
	}{
		{
			name:    "successfully saves valid request",
			request: helpers.CreateAccessRequest("user1", "account1", "manager-group"),
			setupMock: func(m *helpers.MockDynamoDBClient) {
				m.On("PutItem", mock.Anything, mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
					return *input.TableName == "test-table" && input.Item != nil
				}), mock.Anything).Return(&dynamodb.PutItemOutput{}, nil)
			},
			expectError: false,
		},
		{
			name: "fails validation with invalid request",
			request: &models.AccessRequest{
				RequestID: "req-123",
				Status:    "invalid-status",
			},
			setupMock:   func(m *helpers.MockDynamoDBClient) {},
			expectError: true,
			errorMsg:    "invalid request",
		},
		{
			name:    "handles DynamoDB error",
			request: helpers.CreateAccessRequest("user1", "account1", "manager-group"),
			setupMock: func(m *helpers.MockDynamoDBClient) {
				m.On("PutItem", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("dynamodb error"))
			},
			expectError: true,
			errorMsg:    "failed to save request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(helpers.MockDynamoDBClient)
			tt.setupMock(mockClient)

			repo := repository.NewRequestRepository(mockClient, "test-table")
			err := repo.SaveRequest(context.Background(), tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestRequestRepository_GetRequestByID(t *testing.T) {
	tests := []struct {
		name        string
		requestID   string
		setupMock   func(*helpers.MockDynamoDBClient)
		expectError bool
		errorMsg    string
		validate    func(*testing.T, *models.AccessRequest)
	}{
		{
			name:      "successfully retrieves existing request",
			requestID: "req-123",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				request := helpers.CreateAccessRequest("user1", "account1", "manager-group")
				item, _ := attributevalue.MarshalMap(request)
				m.On("GetItem", mock.Anything, mock.MatchedBy(func(input *dynamodb.GetItemInput) bool {
					return *input.TableName == "test-table"
				}), mock.Anything).Return(&dynamodb.GetItemOutput{Item: item}, nil)
			},
			expectError: false,
			validate: func(t *testing.T, request *models.AccessRequest) {
				assert.Equal(t, models.RequestStatusPending, request.Status)
				assert.NotEmpty(t, request.ManagerGroupID)
			},
		},
		{
			name:      "returns error when request not found",
			requestID: "nonexistent",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				m.On("GetItem", mock.Anything, mock.Anything, mock.Anything).
					Return(&dynamodb.GetItemOutput{Item: nil}, nil)
			},
			expectError: true,
			errorMsg:    "request not found",
		},
		{
			name:      "handles DynamoDB error",
			requestID: "req-123",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				m.On("GetItem", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("dynamodb error"))
			},
			expectError: true,
			errorMsg:    "failed to get request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(helpers.MockDynamoDBClient)
			tt.setupMock(mockClient)

			repo := repository.NewRequestRepository(mockClient, "test-table")
			request, err := repo.GetRequestByID(context.Background(), tt.requestID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, request)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, request)
				if tt.validate != nil {
					tt.validate(t, request)
				}
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestRequestRepository_UpdateRequestStatus(t *testing.T) {
	tests := []struct {
		name        string
		requestID   string
		status      models.RequestStatus
		approver    string
		setupMock   func(*helpers.MockDynamoDBClient)
		expectError bool
		errorMsg    string
	}{
		{
			name:      "successfully updates status to approved",
			requestID: "req-123",
			status:    models.RequestStatusApproved,
			approver:  "approver1",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				m.On("UpdateItem", mock.Anything, mock.MatchedBy(func(input *dynamodb.UpdateItemInput) bool {
					return *input.TableName == "test-table"
				}), mock.Anything).Return(&dynamodb.UpdateItemOutput{}, nil)
			},
			expectError: false,
		},
		{
			name:      "successfully updates status to denied",
			requestID: "req-123",
			status:    models.RequestStatusDenied,
			approver:  "denier1",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				m.On("UpdateItem", mock.Anything, mock.Anything, mock.Anything).
					Return(&dynamodb.UpdateItemOutput{}, nil)
			},
			expectError: false,
		},
		{
			name:        "fails with invalid status",
			requestID:   "req-123",
			status:      "invalid",
			approver:    "approver1",
			setupMock:   func(m *helpers.MockDynamoDBClient) {},
			expectError: true,
			errorMsg:    "invalid status",
		},
		{
			name:      "handles DynamoDB error",
			requestID: "req-123",
			status:    models.RequestStatusApproved,
			approver:  "approver1",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				m.On("UpdateItem", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("dynamodb error"))
			},
			expectError: true,
			errorMsg:    "failed to update request status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(helpers.MockDynamoDBClient)
			tt.setupMock(mockClient)

			repo := repository.NewRequestRepository(mockClient, "test-table")
			err := repo.UpdateRequestStatus(context.Background(), tt.requestID, tt.status, tt.approver)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestRequestRepository_ListRequestsByUsername(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		setupMock   func(*helpers.MockDynamoDBClient)
		expectError bool
		expectCount int
	}{
		{
			name:     "successfully lists requests for user",
			username: "user1",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				req1 := helpers.CreateAccessRequest("user1", "account1", "manager-group")
				req2 := helpers.CreateAccessRequest("user1", "account2", "manager-group")
				item1, _ := attributevalue.MarshalMap(req1)
				item2, _ := attributevalue.MarshalMap(req2)
				m.On("Query", mock.Anything, mock.MatchedBy(func(input *dynamodb.QueryInput) bool {
					return *input.IndexName == "username-created_at-index"
				}), mock.Anything).Return(&dynamodb.QueryOutput{
					Items: []map[string]types.AttributeValue{item1, item2},
				}, nil)
			},
			expectError: false,
			expectCount: 2,
		},
		{
			name:     "returns empty list when no requests found",
			username: "user-no-requests",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				m.On("Query", mock.Anything, mock.Anything, mock.Anything).
					Return(&dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{}}, nil)
			},
			expectError: false,
			expectCount: 0,
		},
		{
			name:     "handles DynamoDB error",
			username: "user1",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				m.On("Query", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("dynamodb error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(helpers.MockDynamoDBClient)
			tt.setupMock(mockClient)

			repo := repository.NewRequestRepository(mockClient, "test-table")
			requests, err := repo.ListRequestsByUsername(context.Background(), tt.username)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, requests, tt.expectCount)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestRequestRepository_ListPendingRequests(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*helpers.MockDynamoDBClient)
		expectError bool
		expectCount int
	}{
		{
			name: "successfully lists pending requests",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				req1 := helpers.CreateAccessRequest("user1", "account1", "manager-group")
				req2 := helpers.CreateAccessRequest("user2", "account2", "manager-group")
				req2.Status = models.RequestStatusPartiallyApproved
				item1, _ := attributevalue.MarshalMap(req1)
				item2, _ := attributevalue.MarshalMap(req2)

				// First query for pending requests
				m.On("Query", mock.Anything, mock.MatchedBy(func(input *dynamodb.QueryInput) bool {
					if input.ExpressionAttributeValues == nil {
						return false
					}
					statusAttr, ok := input.ExpressionAttributeValues[":status"]
					if !ok {
						return false
					}
					statusVal, ok := statusAttr.(*types.AttributeValueMemberS)
					if !ok {
						return false
					}
					return statusVal.Value == string(models.RequestStatusPending)
				}), mock.Anything).Return(&dynamodb.QueryOutput{
					Items: []map[string]types.AttributeValue{item1},
				}, nil).Once()

				// Second query for partially approved requests
				m.On("Query", mock.Anything, mock.MatchedBy(func(input *dynamodb.QueryInput) bool {
					if input.ExpressionAttributeValues == nil {
						return false
					}
					statusAttr, ok := input.ExpressionAttributeValues[":status"]
					if !ok {
						return false
					}
					statusVal, ok := statusAttr.(*types.AttributeValueMemberS)
					if !ok {
						return false
					}
					return statusVal.Value == string(models.RequestStatusPartiallyApproved)
				}), mock.Anything).Return(&dynamodb.QueryOutput{
					Items: []map[string]types.AttributeValue{item2},
				}, nil).Once()
			},
			expectError: false,
			expectCount: 2,
		},
		{
			name: "returns empty list when no pending requests",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				// Both queries return empty
				m.On("Query", mock.Anything, mock.Anything, mock.Anything).
					Return(&dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{}}, nil).Twice()
			},
			expectError: false,
			expectCount: 0,
		},
		{
			name: "handles DynamoDB error",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				// First query fails
				m.On("Query", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("dynamodb error")).Once()
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(helpers.MockDynamoDBClient)
			tt.setupMock(mockClient)

			repo := repository.NewRequestRepository(mockClient, "test-table")
			requests, err := repo.ListPendingRequests(context.Background())

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, requests, tt.expectCount)
				// Verify all requests are either pending or partially approved
				for _, req := range requests {
					assert.True(t, req.Status == models.RequestStatusPending || req.Status == models.RequestStatusPartiallyApproved)
				}
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestRequestRepository_ListAllRequests(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*helpers.MockDynamoDBClient)
		expectError bool
		expectCount int
	}{
		{
			name: "successfully lists all requests",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				req1 := helpers.CreateAccessRequest("user1", "account1", "manager-group")
				req2 := helpers.CreateAccessRequest("user2", "account2", "manager-group")
				req2.Status = models.RequestStatusApproved
				item1, _ := attributevalue.MarshalMap(req1)
				item2, _ := attributevalue.MarshalMap(req2)
				m.On("Scan", mock.Anything, mock.MatchedBy(func(input *dynamodb.ScanInput) bool {
					return *input.TableName == "test-table"
				}), mock.Anything).Return(&dynamodb.ScanOutput{
					Items: []map[string]types.AttributeValue{item1, item2},
				}, nil)
			},
			expectError: false,
			expectCount: 2,
		},
		{
			name: "returns empty list when no requests",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				m.On("Scan", mock.Anything, mock.Anything, mock.Anything).
					Return(&dynamodb.ScanOutput{Items: []map[string]types.AttributeValue{}}, nil)
			},
			expectError: false,
			expectCount: 0,
		},
		{
			name: "handles DynamoDB error",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				m.On("Scan", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("dynamodb error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(helpers.MockDynamoDBClient)
			tt.setupMock(mockClient)

			repo := repository.NewRequestRepository(mockClient, "test-table")
			requests, err := repo.ListAllRequests(context.Background())

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, requests, tt.expectCount)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestRequestRepository_DeleteRequest(t *testing.T) {
	tests := []struct {
		name        string
		requestID   string
		setupMock   func(*helpers.MockDynamoDBClient)
		expectError bool
	}{
		{
			name:      "successfully deletes request",
			requestID: "req-123",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				m.On("DeleteItem", mock.Anything, mock.MatchedBy(func(input *dynamodb.DeleteItemInput) bool {
					return *input.TableName == "test-table"
				}), mock.Anything).Return(&dynamodb.DeleteItemOutput{}, nil)
			},
			expectError: false,
		},
		{
			name:      "handles DynamoDB error",
			requestID: "req-123",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				m.On("DeleteItem", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("dynamodb error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(helpers.MockDynamoDBClient)
			tt.setupMock(mockClient)

			repo := repository.NewRequestRepository(mockClient, "test-table")
			err := repo.DeleteRequest(context.Background(), tt.requestID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}
