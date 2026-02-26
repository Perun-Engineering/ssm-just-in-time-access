package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/ssm-access-manager/internal/models"
)

// RequestRepository handles DynamoDB operations for access requests
type RequestRepository struct {
	client    DynamoDBClient
	tableName string
}

// NewRequestRepository creates a new request repository
func NewRequestRepository(client DynamoDBClient, tableName string) *RequestRepository {
	return &RequestRepository{
		client:    client,
		tableName: tableName,
	}
}

// SaveRequest saves an access request to DynamoDB
func (r *RequestRepository) SaveRequest(ctx context.Context, request *models.AccessRequest) error {
	if err := request.Validate(); err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}

	item, err := attributevalue.MarshalMap(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	}

	_, err = r.client.PutItem(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to save request: %w", err)
	}

	return nil
}

// GetRequest retrieves an access request by ID (alias for GetRequestByID)
func (r *RequestRepository) GetRequest(ctx context.Context, requestID string) (*models.AccessRequest, error) {
	return r.GetRequestByID(ctx, requestID)
}

// GetRequestByID retrieves an access request by ID
func (r *RequestRepository) GetRequestByID(ctx context.Context, requestID string) (*models.AccessRequest, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"request_id": &types.AttributeValueMemberS{Value: requestID},
		},
	}

	result, err := r.client.GetItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get request: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("request not found: %s", requestID)
	}

	var request models.AccessRequest
	err = attributevalue.UnmarshalMap(result.Item, &request)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	return &request, nil
}

// UpdateRequestStatus updates the status of an access request
func (r *RequestRepository) UpdateRequestStatus(ctx context.Context, requestID string, status models.RequestStatus, approver string) error {
	if !status.IsValid() {
		return fmt.Errorf("invalid status: %s", status)
	}

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"request_id": &types.AttributeValueMemberS{Value: requestID},
		},
		UpdateExpression: aws.String("SET #status = :status, approver = :approver, updated_at = :updated_at"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":     &types.AttributeValueMemberS{Value: string(status)},
			":approver":   &types.AttributeValueMemberS{Value: approver},
			":updated_at": &types.AttributeValueMemberS{Value: fmt.Sprintf("%d", time.Now().Unix())},
		},
	}

	_, err := r.client.UpdateItem(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to update request status: %w", err)
	}

	return nil
}

// ListRequestsByUsername retrieves all requests for a specific username
func (r *RequestRepository) ListRequestsByUsername(ctx context.Context, username string) ([]*models.AccessRequest, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("username-created_at-index"),
		KeyConditionExpression: aws.String("username = :username"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":username": &types.AttributeValueMemberS{Value: username},
		},
		ScanIndexForward: aws.Bool(false), // Sort by created_at descending
	}

	result, err := r.client.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list requests by username: %w", err)
	}

	var requests []*models.AccessRequest
	err = attributevalue.UnmarshalListOfMaps(result.Items, &requests)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal requests: %w", err)
	}

	return requests, nil
}

// ListRequestsByUser retrieves all requests for a specific user ID (alias for ListRequestsByUsername)
func (r *RequestRepository) ListRequestsByUser(ctx context.Context, userID string) ([]*models.AccessRequest, error) {
	return r.ListRequestsByUsername(ctx, userID)
}

// ListPendingRequests retrieves all pending and partially approved access requests
func (r *RequestRepository) ListPendingRequests(ctx context.Context) ([]*models.AccessRequest, error) {
	// Query for pending requests
	pendingInput := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("status-created_at-index"),
		KeyConditionExpression: aws.String("#status = :status"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: string(models.RequestStatusPending)},
		},
		ScanIndexForward: aws.Bool(false), // Sort by created_at descending
	}

	pendingResult, err := r.client.Query(ctx, pendingInput)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending requests: %w", err)
	}

	var pendingRequests []*models.AccessRequest
	err = attributevalue.UnmarshalListOfMaps(pendingResult.Items, &pendingRequests)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal pending requests: %w", err)
	}

	// Query for partially approved requests
	partialInput := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("status-created_at-index"),
		KeyConditionExpression: aws.String("#status = :status"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: string(models.RequestStatusPartiallyApproved)},
		},
		ScanIndexForward: aws.Bool(false), // Sort by created_at descending
	}

	partialResult, err := r.client.Query(ctx, partialInput)
	if err != nil {
		return nil, fmt.Errorf("failed to list partially approved requests: %w", err)
	}

	var partialRequests []*models.AccessRequest
	err = attributevalue.UnmarshalListOfMaps(partialResult.Items, &partialRequests)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal partially approved requests: %w", err)
	}

	// Combine both lists
	allRequests := append(pendingRequests, partialRequests...)

	// Sort by created_at descending (most recent first)
	for i := 0; i < len(allRequests)-1; i++ {
		for j := i + 1; j < len(allRequests); j++ {
			if allRequests[i].CreatedAt.Before(allRequests[j].CreatedAt) {
				allRequests[i], allRequests[j] = allRequests[j], allRequests[i]
			}
		}
	}

	return allRequests, nil
}

// ListAllRequests retrieves all access requests (all statuses)
func (r *RequestRepository) ListAllRequests(ctx context.Context) ([]*models.AccessRequest, error) {
	input := &dynamodb.ScanInput{
		TableName: aws.String(r.tableName),
	}

	result, err := r.client.Scan(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list all requests: %w", err)
	}

	var requests []*models.AccessRequest
	err = attributevalue.UnmarshalListOfMaps(result.Items, &requests)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal requests: %w", err)
	}

	// Sort by created_at descending (most recent first)
	// Since Scan doesn't support sorting, we do it in memory
	for i := 0; i < len(requests)-1; i++ {
		for j := i + 1; j < len(requests); j++ {
			if requests[i].CreatedAt.Before(requests[j].CreatedAt) {
				requests[i], requests[j] = requests[j], requests[i]
			}
		}
	}

	return requests, nil
}

// UpdateRequest updates an access request
func (r *RequestRepository) UpdateRequest(ctx context.Context, request *models.AccessRequest) error {
	if err := request.Validate(); err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}

	request.UpdatedAt = time.Now()

	item, err := attributevalue.MarshalMap(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	}

	_, err = r.client.PutItem(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to update request: %w", err)
	}

	return nil
}

// DeleteRequest deletes an access request
func (r *RequestRepository) DeleteRequest(ctx context.Context, requestID string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"request_id": &types.AttributeValueMemberS{Value: requestID},
		},
	}

	_, err := r.client.DeleteItem(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete request: %w", err)
	}

	return nil
}
