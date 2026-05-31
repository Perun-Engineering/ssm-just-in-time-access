package services_test

import (
	"testing"

	"github.com/ssm-access-manager/internal/validation"
	"github.com/stretchr/testify/assert"
)

// TestSSMDocumentService_GenerateDocumentContent tests document content generation
func TestSSMDocumentService_GenerateDocumentContent(t *testing.T) {
	// Note: This is a simplified test since the actual service requires AWS SDK mocks
	// which would be complex to set up. In a production environment, you would:
	// 1. Create interfaces for AWS SDK clients
	// 2. Mock the SSM client
	// 3. Test the full CreateDocument flow

	// For now, we test that the service can be instantiated with the correct dependencies
	nameGenerator := validation.NewDocumentNameGeneratorWithPrefix("PF")

	assert.NotNil(t, nameGenerator)

	// Test document name generation
	documentName := nameGenerator.GenerateName("testuser", "prod-db-01", 5432)
	assert.NotEmpty(t, documentName)
	assert.Contains(t, documentName, "PF")
	assert.Contains(t, documentName, "testuser")
}

// TestSSMDocumentService_DocumentNameGeneration tests name generation with custom prefix
func TestSSMDocumentService_DocumentNameGeneration_CustomPrefix(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		username string
		host     string
		port     int
	}{
		{
			name:     "default prefix",
			prefix:   "PF",
			username: "john.doe",
			host:     "prod-db-01",
			port:     5432,
		},
		{
			name:     "custom prefix",
			prefix:   "CUSTOM",
			username: "jane.smith",
			host:     "staging-api",
			port:     8080,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nameGenerator := validation.NewDocumentNameGeneratorWithPrefix(tt.prefix)
			documentName := nameGenerator.GenerateName(tt.username, tt.host, tt.port)

			assert.NotEmpty(t, documentName)
			assert.Contains(t, documentName, tt.prefix)
		})
	}
}

// TestSSMDocumentService_DocumentNameValidation tests document name validation
func TestSSMDocumentService_DocumentNameValidation(t *testing.T) {
	nameGenerator := validation.NewDocumentNameGeneratorWithPrefix("PF")

	tests := []struct {
		name          string
		documentName  string
		shouldBeValid bool
	}{
		{
			name:          "valid document name",
			documentName:  "PF-testuser-prod-db-01-5432",
			shouldBeValid: true,
		},
		{
			name:          "empty document name",
			documentName:  "",
			shouldBeValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := nameGenerator.ValidateDocumentName(tt.documentName)

			if tt.shouldBeValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// TestSSMDocumentService_ParseDocumentName tests parsing document names
func TestSSMDocumentService_ParseDocumentName(t *testing.T) {
	nameGenerator := validation.NewDocumentNameGeneratorWithPrefix("PF")

	// Generate a document name
	username := "testuser"
	host := "prod-db-01"
	port := 5432
	documentName := nameGenerator.GenerateName(username, host, port)

	// Parse it back
	parsedUsername, parsedHost, parsedPort, err := nameGenerator.ParseDocumentName(documentName)

	assert.NoError(t, err)
	assert.Equal(t, username, parsedUsername)
	assert.Equal(t, host, parsedHost)
	assert.Equal(t, port, parsedPort)
}

// TestSSMDocumentService_SanitizeSpecialCharacters tests special character handling
func TestSSMDocumentService_SanitizeSpecialCharacters(t *testing.T) {
	nameGenerator := validation.NewDocumentNameGeneratorWithPrefix("PF")

	tests := []struct {
		name     string
		username string
		host     string
	}{
		{
			name:     "username with dots",
			username: "john.doe",
			host:     "prod-db-01",
		},
		{
			name:     "host with special chars",
			username: "user",
			host:     "prod_db_01.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			documentName := nameGenerator.GenerateName(tt.username, tt.host, 5432)

			// Document name should not contain certain special characters
			assert.NotEmpty(t, documentName)
			// The name generator should handle special characters appropriately
			err := nameGenerator.ValidateDocumentName(documentName)
			assert.NoError(t, err)
		})
	}
}

// Note: Full integration tests for SSMDocumentService would require:
// 1. Mocking AWS SSM Client
// 2. Mocking RoleAssumer
// 3. Mocking DocumentRepository
// 4. Testing CreateDocument, DeleteDocument, ExtendDocumentExpiration flows
//
// These tests focus on the DocumentNameGenerator which is a critical component
// of the SSMDocumentService. Full service tests would be added in integration tests
// where we can use AWS SDK mocks or localstack.
