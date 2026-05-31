package repository

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/ssm-access-manager/internal/models"
)

// DynamoDBClient interface for testing
type DynamoDBClient interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
}

// ApprovalGroupRepository handles DynamoDB operations for approval groups
type ApprovalGroupRepository struct {
	client    DynamoDBClient
	tableName string
}

// NewApprovalGroupRepository creates a new approval group repository
func NewApprovalGroupRepository(client DynamoDBClient, tableName string) *ApprovalGroupRepository {
	return &ApprovalGroupRepository{
		client:    client,
		tableName: tableName,
	}
}

// SaveGroup saves an approval group to DynamoDB
func (r *ApprovalGroupRepository) SaveGroup(ctx context.Context, group *models.ApprovalGroup) error {
	// Validate the group before saving
	if err := group.Validate(); err != nil {
		return fmt.Errorf("invalid approval group: %w", err)
	}

	// Marshal the group to DynamoDB attribute values
	item, err := attributevalue.MarshalMap(group)
	if err != nil {
		return fmt.Errorf("failed to marshal approval group: %w", err)
	}

	// Put the item in DynamoDB
	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to save approval group: %w", err)
	}

	return nil
}

// GetGroup retrieves an approval group by ID
func (r *ApprovalGroupRepository) GetGroup(ctx context.Context, groupID string) (*models.ApprovalGroup, error) {
	result, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"group_id": &types.AttributeValueMemberS{Value: groupID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get approval group: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("approval group not found: %s", groupID)
	}

	var group models.ApprovalGroup
	err = attributevalue.UnmarshalMap(result.Item, &group)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal approval group: %w", err)
	}

	return &group, nil
}

// ListAllGroups retrieves all approval groups
func (r *ApprovalGroupRepository) ListAllGroups(ctx context.Context) ([]*models.ApprovalGroup, error) {
	result, err := r.client.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(r.tableName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list approval groups: %w", err)
	}

	groups := make([]*models.ApprovalGroup, 0, len(result.Items))
	for _, item := range result.Items {
		var group models.ApprovalGroup
		err = attributevalue.UnmarshalMap(item, &group)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal approval group: %w", err)
		}
		groups = append(groups, &group)
	}

	return groups, nil
}

// ListGroupsByType retrieves all approval groups of a specific type
func (r *ApprovalGroupRepository) ListGroupsByType(ctx context.Context, groupType models.ApprovalGroupType) ([]*models.ApprovalGroup, error) {
	result, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("group_type-group_name-index"),
		KeyConditionExpression: aws.String("group_type = :group_type"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":group_type": &types.AttributeValueMemberS{Value: string(groupType)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list groups by type: %w", err)
	}

	groups := make([]*models.ApprovalGroup, 0, len(result.Items))
	for _, item := range result.Items {
		var group models.ApprovalGroup
		err = attributevalue.UnmarshalMap(item, &group)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal approval group: %w", err)
		}
		groups = append(groups, &group)
	}

	return groups, nil
}

// ListActiveManagerGroups retrieves all active manager groups
func (r *ApprovalGroupRepository) ListActiveManagerGroups(ctx context.Context) ([]*models.ApprovalGroup, error) {
	result, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("group_type-group_name-index"),
		KeyConditionExpression: aws.String("group_type = :group_type"),
		FilterExpression:       aws.String("active = :active"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":group_type": &types.AttributeValueMemberS{Value: string(models.ApprovalGroupTypeManager)},
			":active":     &types.AttributeValueMemberBOOL{Value: true},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list active manager groups: %w", err)
	}

	groups := make([]*models.ApprovalGroup, 0, len(result.Items))
	for _, item := range result.Items {
		var group models.ApprovalGroup
		err = attributevalue.UnmarshalMap(item, &group)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal approval group: %w", err)
		}
		groups = append(groups, &group)
	}

	return groups, nil
}

// GetSecurityGroup retrieves the security group (there should only be one)
func (r *ApprovalGroupRepository) GetSecurityGroup(ctx context.Context) (*models.ApprovalGroup, error) {
	result, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("group_type-group_name-index"),
		KeyConditionExpression: aws.String("group_type = :group_type"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":group_type": &types.AttributeValueMemberS{Value: string(models.ApprovalGroupTypeSecurity)},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get security group: %w", err)
	}

	if len(result.Items) == 0 {
		return nil, fmt.Errorf("security group not configured")
	}

	var group models.ApprovalGroup
	err = attributevalue.UnmarshalMap(result.Items[0], &group)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal security group: %w", err)
	}

	return &group, nil
}

// UpdateGroup updates an existing approval group
func (r *ApprovalGroupRepository) UpdateGroup(ctx context.Context, group *models.ApprovalGroup) error {
	// Validate the group before updating
	if err := group.Validate(); err != nil {
		return fmt.Errorf("invalid approval group: %w", err)
	}

	// Marshal the group to DynamoDB attribute values
	item, err := attributevalue.MarshalMap(group)
	if err != nil {
		return fmt.Errorf("failed to marshal approval group: %w", err)
	}

	// Update the item in DynamoDB
	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to update approval group: %w", err)
	}

	return nil
}

// DeleteGroup deletes an approval group
func (r *ApprovalGroupRepository) DeleteGroup(ctx context.Context, groupID string) error {
	_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"group_id": &types.AttributeValueMemberS{Value: groupID},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete approval group: %w", err)
	}

	return nil
}
