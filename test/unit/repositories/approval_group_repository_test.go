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

func TestApprovalGroupRepository_SaveGroup(t *testing.T) {
	tests := []struct {
		name        string
		group       *models.ApprovalGroup
		setupMock   func(*helpers.MockDynamoDBClient)
		expectError bool
		errorMsg    string
	}{
		{
			name:  "successfully saves valid security group",
			group: helpers.CreateSecurityGroup(),
			setupMock: func(m *helpers.MockDynamoDBClient) {
				m.On("PutItem", mock.Anything, mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
					return *input.TableName == "test-table" && input.Item != nil
				}), mock.Anything).Return(&dynamodb.PutItemOutput{}, nil)
			},
			expectError: false,
		},
		{
			name:  "successfully saves valid manager group",
			group: helpers.CreateManagerGroup("eng-team", "Engineering Team"),
			setupMock: func(m *helpers.MockDynamoDBClient) {
				m.On("PutItem", mock.Anything, mock.Anything, mock.Anything).
					Return(&dynamodb.PutItemOutput{}, nil)
			},
			expectError: false,
		},
		{
			name: "fails validation with invalid group type",
			group: &models.ApprovalGroup{
				GroupID:     "test-group",
				GroupType:   "invalid",
				GroupName:   "Test Group",
				SlackHandle: "@test",
				Active:      true,
			},
			setupMock:   func(m *helpers.MockDynamoDBClient) {},
			expectError: true,
			errorMsg:    "invalid approval group",
		},
		{
			name: "fails validation with empty required fields",
			group: &models.ApprovalGroup{
				GroupType: models.ApprovalGroupTypeSecurity,
			},
			setupMock:   func(m *helpers.MockDynamoDBClient) {},
			expectError: true,
			errorMsg:    "invalid approval group",
		},
		{
			name:  "handles DynamoDB error",
			group: helpers.CreateSecurityGroup(),
			setupMock: func(m *helpers.MockDynamoDBClient) {
				m.On("PutItem", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("dynamodb error"))
			},
			expectError: true,
			errorMsg:    "failed to save approval group",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(helpers.MockDynamoDBClient)
			tt.setupMock(mockClient)

			repo := repository.NewApprovalGroupRepository(mockClient, "test-table")
			err := repo.SaveGroup(context.Background(), tt.group)

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

func TestApprovalGroupRepository_GetGroup(t *testing.T) {
	tests := []struct {
		name        string
		groupID     string
		setupMock   func(*helpers.MockDynamoDBClient)
		expectError bool
		errorMsg    string
		validate    func(*testing.T, *models.ApprovalGroup)
	}{
		{
			name:    "successfully retrieves existing group",
			groupID: "security-group",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				group := helpers.CreateSecurityGroup()
				item, _ := attributevalue.MarshalMap(group)
				m.On("GetItem", mock.Anything, mock.MatchedBy(func(input *dynamodb.GetItemInput) bool {
					return *input.TableName == "test-table"
				}), mock.Anything).Return(&dynamodb.GetItemOutput{Item: item}, nil)
			},
			expectError: false,
			validate: func(t *testing.T, group *models.ApprovalGroup) {
				assert.Equal(t, models.ApprovalGroupTypeSecurity, group.GroupType)
				assert.True(t, group.Active)
			},
		},
		{
			name:    "returns error when group not found",
			groupID: "nonexistent",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				m.On("GetItem", mock.Anything, mock.Anything, mock.Anything).
					Return(&dynamodb.GetItemOutput{Item: nil}, nil)
			},
			expectError: true,
			errorMsg:    "approval group not found",
		},
		{
			name:    "handles DynamoDB error",
			groupID: "test-group",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				m.On("GetItem", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("dynamodb error"))
			},
			expectError: true,
			errorMsg:    "failed to get approval group",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(helpers.MockDynamoDBClient)
			tt.setupMock(mockClient)

			repo := repository.NewApprovalGroupRepository(mockClient, "test-table")
			group, err := repo.GetGroup(context.Background(), tt.groupID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, group)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, group)
				if tt.validate != nil {
					tt.validate(t, group)
				}
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestApprovalGroupRepository_ListGroupsByType(t *testing.T) {
	tests := []struct {
		name        string
		groupType   models.ApprovalGroupType
		setupMock   func(*helpers.MockDynamoDBClient)
		expectError bool
		expectCount int
	}{
		{
			name:      "successfully lists security groups",
			groupType: models.ApprovalGroupTypeSecurity,
			setupMock: func(m *helpers.MockDynamoDBClient) {
				group := helpers.CreateSecurityGroup()
				item, _ := attributevalue.MarshalMap(group)
				m.On("Query", mock.Anything, mock.MatchedBy(func(input *dynamodb.QueryInput) bool {
					return *input.IndexName == "group_type-group_name-index"
				}), mock.Anything).Return(&dynamodb.QueryOutput{
					Items: []map[string]types.AttributeValue{item},
				}, nil)
			},
			expectError: false,
			expectCount: 1,
		},
		{
			name:      "successfully lists manager groups",
			groupType: models.ApprovalGroupTypeManager,
			setupMock: func(m *helpers.MockDynamoDBClient) {
				group1 := helpers.CreateManagerGroup("eng", "Engineering")
				group2 := helpers.CreateManagerGroup("ops", "Operations")
				item1, _ := attributevalue.MarshalMap(group1)
				item2, _ := attributevalue.MarshalMap(group2)
				m.On("Query", mock.Anything, mock.Anything, mock.Anything).
					Return(&dynamodb.QueryOutput{
						Items: []map[string]types.AttributeValue{item1, item2},
					}, nil)
			},
			expectError: false,
			expectCount: 2,
		},
		{
			name:      "returns empty list when no groups found",
			groupType: models.ApprovalGroupTypeSecurity,
			setupMock: func(m *helpers.MockDynamoDBClient) {
				m.On("Query", mock.Anything, mock.Anything, mock.Anything).
					Return(&dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{}}, nil)
			},
			expectError: false,
			expectCount: 0,
		},
		{
			name:      "handles DynamoDB error",
			groupType: models.ApprovalGroupTypeSecurity,
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

			repo := repository.NewApprovalGroupRepository(mockClient, "test-table")
			groups, err := repo.ListGroupsByType(context.Background(), tt.groupType)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, groups, tt.expectCount)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestApprovalGroupRepository_GetSecurityGroup(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*helpers.MockDynamoDBClient)
		expectError bool
		errorMsg    string
	}{
		{
			name: "successfully retrieves security group",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				group := helpers.CreateSecurityGroup()
				item, _ := attributevalue.MarshalMap(group)
				m.On("Query", mock.Anything, mock.Anything, mock.Anything).
					Return(&dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{item}}, nil)
			},
			expectError: false,
		},
		{
			name: "returns error when security group not configured",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				m.On("Query", mock.Anything, mock.Anything, mock.Anything).
					Return(&dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{}}, nil)
			},
			expectError: true,
			errorMsg:    "security group not configured",
		},
		{
			name: "handles DynamoDB error",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				m.On("Query", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("dynamodb error"))
			},
			expectError: true,
			errorMsg:    "failed to get security group",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(helpers.MockDynamoDBClient)
			tt.setupMock(mockClient)

			repo := repository.NewApprovalGroupRepository(mockClient, "test-table")
			group, err := repo.GetSecurityGroup(context.Background())

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, group)
				assert.Equal(t, models.ApprovalGroupTypeSecurity, group.GroupType)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestApprovalGroupRepository_ListActiveManagerGroups(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*helpers.MockDynamoDBClient)
		expectError bool
		expectCount int
	}{
		{
			name: "successfully lists active manager groups",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				group1 := helpers.CreateManagerGroup("eng", "Engineering")
				group2 := helpers.CreateManagerGroup("ops", "Operations")
				item1, _ := attributevalue.MarshalMap(group1)
				item2, _ := attributevalue.MarshalMap(group2)
				m.On("Query", mock.Anything, mock.MatchedBy(func(input *dynamodb.QueryInput) bool {
					return input.FilterExpression != nil && *input.FilterExpression == "active = :active"
				}), mock.Anything).Return(&dynamodb.QueryOutput{
					Items: []map[string]types.AttributeValue{item1, item2},
				}, nil)
			},
			expectError: false,
			expectCount: 2,
		},
		{
			name: "returns empty list when no active groups",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				m.On("Query", mock.Anything, mock.Anything, mock.Anything).
					Return(&dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{}}, nil)
			},
			expectError: false,
			expectCount: 0,
		},
		{
			name: "handles DynamoDB error",
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

			repo := repository.NewApprovalGroupRepository(mockClient, "test-table")
			groups, err := repo.ListActiveManagerGroups(context.Background())

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, groups, tt.expectCount)
				for _, group := range groups {
					assert.True(t, group.Active)
					assert.Equal(t, models.ApprovalGroupTypeManager, group.GroupType)
				}
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestApprovalGroupRepository_UpdateGroup(t *testing.T) {
	tests := []struct {
		name        string
		group       *models.ApprovalGroup
		setupMock   func(*helpers.MockDynamoDBClient)
		expectError bool
		errorMsg    string
	}{
		{
			name:  "successfully updates valid group",
			group: helpers.CreateSecurityGroup(),
			setupMock: func(m *helpers.MockDynamoDBClient) {
				m.On("PutItem", mock.Anything, mock.Anything, mock.Anything).
					Return(&dynamodb.PutItemOutput{}, nil)
			},
			expectError: false,
		},
		{
			name: "fails validation with invalid group",
			group: &models.ApprovalGroup{
				GroupType: "invalid",
			},
			setupMock:   func(m *helpers.MockDynamoDBClient) {},
			expectError: true,
			errorMsg:    "invalid approval group",
		},
		{
			name:  "handles DynamoDB error",
			group: helpers.CreateSecurityGroup(),
			setupMock: func(m *helpers.MockDynamoDBClient) {
				m.On("PutItem", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("dynamodb error"))
			},
			expectError: true,
			errorMsg:    "failed to update approval group",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(helpers.MockDynamoDBClient)
			tt.setupMock(mockClient)

			repo := repository.NewApprovalGroupRepository(mockClient, "test-table")
			err := repo.UpdateGroup(context.Background(), tt.group)

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

func TestApprovalGroupRepository_DeleteGroup(t *testing.T) {
	tests := []struct {
		name        string
		groupID     string
		setupMock   func(*helpers.MockDynamoDBClient)
		expectError bool
	}{
		{
			name:    "successfully deletes group",
			groupID: "test-group",
			setupMock: func(m *helpers.MockDynamoDBClient) {
				m.On("DeleteItem", mock.Anything, mock.MatchedBy(func(input *dynamodb.DeleteItemInput) bool {
					return *input.TableName == "test-table"
				}), mock.Anything).Return(&dynamodb.DeleteItemOutput{}, nil)
			},
			expectError: false,
		},
		{
			name:    "handles DynamoDB error",
			groupID: "test-group",
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

			repo := repository.NewApprovalGroupRepository(mockClient, "test-table")
			err := repo.DeleteGroup(context.Background(), tt.groupID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}
