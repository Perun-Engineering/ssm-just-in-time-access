package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/google/uuid"
	awshelper "github.com/ssm-access-manager/pkg/aws"
	
	"github.com/ssm-access-manager/internal/models"
	"github.com/ssm-access-manager/internal/repository"
	"github.com/ssm-access-manager/internal/validation"
)

// SSMDocumentService handles SSM document lifecycle management
type SSMDocumentService struct {
	documentRepo  *repository.DocumentRepository
	roleAssumer   *awshelper.RoleAssumer
	nameGenerator *validation.DocumentNameGenerator
}

// NewSSMDocumentService creates a new SSM document service
func NewSSMDocumentService(
	documentRepo *repository.DocumentRepository,
	roleAssumer *awshelper.RoleAssumer,
	nameGenerator *validation.DocumentNameGenerator,
) *SSMDocumentService {
	return &SSMDocumentService{
		documentRepo:  documentRepo,
		roleAssumer:   roleAssumer,
		nameGenerator: nameGenerator,
	}
}

// SSMDocumentContent represents the content structure of an SSM document
type SSMDocumentContent struct {
	SchemaVersion string                 `json:"schemaVersion"`
	Description   string                 `json:"description"`
	SessionType   string                 `json:"sessionType"`
	Parameters    map[string]interface{} `json:"parameters"`
	Properties    map[string]interface{} `json:"properties"`
}

// CreateDocument creates an SSM document in the target account
func (s *SSMDocumentService) CreateDocument(
	ctx context.Context,
	request *models.AccessRequest,
	accountID, region string,
) (*models.SSMDocument, error) {
	// Check for existing active document
	existingDoc, err := s.CheckExistingDocument(ctx, request.Username, request.Host, request.Port, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing document: %w", err)
	}

	if existingDoc != nil && existingDoc.IsActive() {
		// Extend expiration if new expiration is later
		if request.ExpirationDate.After(existingDoc.ExpiresAt) {
			return s.ExtendDocumentExpiration(ctx, existingDoc, request.ExpirationDate)
		}
		return existingDoc, nil
	}

	// Generate document name
	documentName := s.nameGenerator.GenerateName(request.Username, request.Host, request.Port)

	// Validate document name
	if err := s.nameGenerator.ValidateDocumentName(documentName); err != nil {
		return nil, fmt.Errorf("invalid document name: %w", err)
	}

	// Generate document content
	content, err := s.generateDocumentContent(request.Username, request.Host, request.Port)
	if err != nil {
		return nil, fmt.Errorf("failed to generate document content: %w", err)
	}

	// Get SSM client with assumed role
	ssmClient, err := s.roleAssumer.GetSSMClient(ctx, accountID, "SSMDocumentManagerRole", region)
	if err != nil {
		return nil, fmt.Errorf("failed to get SSM client: %w", err)
	}

	// Create the SSM document
	tags := []types.Tag{
		{
			Key:   aws.String("ExpiresAt"),
			Value: aws.String(request.ExpirationDate.Format(time.RFC3339)),
		},
		{
			Key:   aws.String("Username"),
			Value: aws.String(request.Username),
		},
		{
			Key:   aws.String("ManagedBy"),
			Value: aws.String("SSM-Access-Manager"),
		},
		{
			Key:   aws.String("RequestID"),
			Value: aws.String(request.RequestID),
		},
	}

	input := &ssm.CreateDocumentInput{
		Name:           aws.String(documentName),
		Content:        aws.String(content),
		DocumentType:   types.DocumentTypeSession,
		DocumentFormat: types.DocumentFormatJson,
		Tags:           tags,
	}

	_, err = ssmClient.CreateDocument(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSM document: %w", err)
	}

	// Create document metadata
	now := time.Now()
	document := &models.SSMDocument{
		DocumentID:   uuid.New().String(),
		DocumentName: documentName,
		AccountID:    accountID,
		Username:     request.Username,
		Host:         request.Host,
		Port:         request.Port,
		RequestID:    request.RequestID,
		CreatedAt:    now,
		ExpiresAt:    request.ExpirationDate,
		Status:       models.DocumentStatusActive,
		Region:       region,
		UpdatedAt:    now,
	}

	// Save document metadata
	err = s.documentRepo.SaveDocument(ctx, document)
	if err != nil {
		// Try to delete the SSM document if metadata save fails
		_ = s.deleteSSMDocument(ctx, ssmClient, documentName)
		return nil, fmt.Errorf("failed to save document metadata: %w", err)
	}

	return document, nil
}

// DeleteDocument deletes an SSM document from the target account
func (s *SSMDocumentService) DeleteDocument(ctx context.Context, document *models.SSMDocument) error {
	// Get SSM client with assumed role
	ssmClient, err := s.roleAssumer.GetSSMClient(ctx, document.AccountID, "SSMDocumentManagerRole", document.Region)
	if err != nil {
		return fmt.Errorf("failed to get SSM client: %w", err)
	}

	// Delete the SSM document
	err = s.deleteSSMDocument(ctx, ssmClient, document.DocumentName)
	if err != nil {
		return fmt.Errorf("failed to delete SSM document: %w", err)
	}

	// Update document status
	err = s.documentRepo.UpdateDocumentStatus(ctx, document.DocumentID, models.DocumentStatusDeleted)
	if err != nil {
		return fmt.Errorf("failed to update document status: %w", err)
	}

	return nil
}

// CheckExistingDocument checks if an active document already exists
func (s *SSMDocumentService) CheckExistingDocument(
	ctx context.Context,
	username, host string,
	port int,
	accountID string,
) (*models.SSMDocument, error) {
	documentName := s.nameGenerator.GenerateName(username, host, port)
	
	document, err := s.documentRepo.GetDocumentByName(ctx, documentName, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document by name: %w", err)
	}

	return document, nil
}

// ExtendDocumentExpiration extends the expiration date of an existing document
func (s *SSMDocumentService) ExtendDocumentExpiration(
	ctx context.Context,
	document *models.SSMDocument,
	newExpiration time.Time,
) (*models.SSMDocument, error) {
	// Get SSM client with assumed role
	ssmClient, err := s.roleAssumer.GetSSMClient(ctx, document.AccountID, "SSMDocumentManagerRole", document.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to get SSM client: %w", err)
	}

	// Update the ExpiresAt tag
	input := &ssm.AddTagsToResourceInput{
		ResourceType: types.ResourceTypeForTaggingDocument,
		ResourceId:   aws.String(document.DocumentName),
		Tags: []types.Tag{
			{
				Key:   aws.String("ExpiresAt"),
				Value: aws.String(newExpiration.Format(time.RFC3339)),
			},
		},
	}

	_, err = ssmClient.AddTagsToResource(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to update document tags: %w", err)
	}

	// Update document metadata
	document.ExpiresAt = newExpiration
	document.UpdatedAt = time.Now()

	err = s.documentRepo.SaveDocument(ctx, document)
	if err != nil {
		return nil, fmt.Errorf("failed to update document metadata: %w", err)
	}

	return document, nil
}

// generateDocumentContent generates the JSON content for an SSM document
func (s *SSMDocumentService) generateDocumentContent(username, host string, port int) (string, error) {
	content := SSMDocumentContent{
		SchemaVersion: "1.0",
		Description:   fmt.Sprintf("Port forwarding session for %s to %s:%d", username, host, port),
		SessionType:   "Port",
		Parameters: map[string]interface{}{
			"portNumber": map[string]interface{}{
				"type":        "String",
				"description": "Port to forward",
				"default":     fmt.Sprintf("%d", port),
			},
			"localPortNumber": map[string]interface{}{
				"type":        "String",
				"description": "Local port number",
				"default":     fmt.Sprintf("%d", port),
			},
		},
		Properties: map[string]interface{}{
			"type":            "LocalPortForwarding",
			"portNumber":      fmt.Sprintf("%d", port),
			"localPortNumber": fmt.Sprintf("%d", port),
		},
	}

	contentJSON, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal document content: %w", err)
	}

	return string(contentJSON), nil
}

// deleteSSMDocument deletes an SSM document (helper method)
func (s *SSMDocumentService) deleteSSMDocument(ctx context.Context, ssmClient *ssm.Client, documentName string) error {
	input := &ssm.DeleteDocumentInput{
		Name: aws.String(documentName),
	}

	_, err := ssmClient.DeleteDocument(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete document %s: %w", documentName, err)
	}

	return nil
}
