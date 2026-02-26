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

// DocumentRepository handles DynamoDB operations for SSM documents
type DocumentRepository struct {
	client    *dynamodb.Client
	tableName string
}

// NewDocumentRepository creates a new document repository
func NewDocumentRepository(client *dynamodb.Client, tableName string) *DocumentRepository {
	return &DocumentRepository{
		client:    client,
		tableName: tableName,
	}
}

// SaveDocument saves an SSM document to DynamoDB
func (r *DocumentRepository) SaveDocument(ctx context.Context, document *models.SSMDocument) error {
	if err := document.Validate(); err != nil {
		return fmt.Errorf("invalid document: %w", err)
	}

	item, err := attributevalue.MarshalMap(document)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	}

	_, err = r.client.PutItem(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to save document: %w", err)
	}

	return nil
}

// GetDocumentByID retrieves an SSM document by ID
func (r *DocumentRepository) GetDocumentByID(ctx context.Context, documentID string) (*models.SSMDocument, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"document_id": &types.AttributeValueMemberS{Value: documentID},
		},
	}

	result, err := r.client.GetItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("document not found: %s", documentID)
	}

	var document models.SSMDocument
	err = attributevalue.UnmarshalMap(result.Item, &document)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal document: %w", err)
	}

	return &document, nil
}

// GetDocumentByName retrieves an SSM document by name and account ID
func (r *DocumentRepository) GetDocumentByName(ctx context.Context, documentName, accountID string) (*models.SSMDocument, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("account_id-document_name-index"),
		KeyConditionExpression: aws.String("account_id = :account_id AND document_name = :document_name"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":account_id":    &types.AttributeValueMemberS{Value: accountID},
			":document_name": &types.AttributeValueMemberS{Value: documentName},
		},
	}

	result, err := r.client.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get document by name: %w", err)
	}

	if len(result.Items) == 0 {
		return nil, nil // Document not found
	}

	var document models.SSMDocument
	err = attributevalue.UnmarshalMap(result.Items[0], &document)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal document: %w", err)
	}

	return &document, nil
}

// UpdateDocumentStatus updates the status of an SSM document
func (r *DocumentRepository) UpdateDocumentStatus(ctx context.Context, documentID string, status models.DocumentStatus) error {
	if !status.IsValid() {
		return fmt.Errorf("invalid status: %s", status)
	}

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"document_id": &types.AttributeValueMemberS{Value: documentID},
		},
		UpdateExpression: aws.String("SET #status = :status, updated_at = :updated_at"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":     &types.AttributeValueMemberS{Value: string(status)},
			":updated_at": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
		},
	}

	_, err := r.client.UpdateItem(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to update document status: %w", err)
	}

	return nil
}

// GetExpiredDocuments retrieves all expired documents
func (r *DocumentRepository) GetExpiredDocuments(ctx context.Context) ([]*models.SSMDocument, error) {
	now := time.Now().Format(time.RFC3339)

	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("status-expires_at-index"),
		KeyConditionExpression: aws.String("#status = :status AND expires_at < :now"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: string(models.DocumentStatusActive)},
			":now":    &types.AttributeValueMemberS{Value: now},
		},
	}

	result, err := r.client.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired documents: %w", err)
	}

	var documents []*models.SSMDocument
	err = attributevalue.UnmarshalListOfMaps(result.Items, &documents)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal documents: %w", err)
	}

	return documents, nil
}

// ListDocumentsByUsername retrieves all documents for a specific username
func (r *DocumentRepository) ListDocumentsByUsername(ctx context.Context, username string) ([]*models.SSMDocument, error) {
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
		return nil, fmt.Errorf("failed to list documents by username: %w", err)
	}

	var documents []*models.SSMDocument
	err = attributevalue.UnmarshalListOfMaps(result.Items, &documents)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal documents: %w", err)
	}

	return documents, nil
}

// DeleteDocument deletes an SSM document
func (r *DocumentRepository) DeleteDocument(ctx context.Context, documentID string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"document_id": &types.AttributeValueMemberS{Value: documentID},
		},
	}

	_, err := r.client.DeleteItem(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}

// GetDocumentByRequestID retrieves an SSM document by request ID
func (r *DocumentRepository) GetDocumentByRequestID(ctx context.Context, requestID string) (*models.SSMDocument, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("request_id-index"),
		KeyConditionExpression: aws.String("request_id = :request_id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":request_id": &types.AttributeValueMemberS{Value: requestID},
		},
	}

	result, err := r.client.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to query document by request ID: %w", err)
	}

	if len(result.Items) == 0 {
		return nil, fmt.Errorf("document not found for request ID: %s", requestID)
	}

	var document models.SSMDocument
	err = attributevalue.UnmarshalMap(result.Items[0], &document)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal document: %w", err)
	}

	return &document, nil
}
