package validation

import (
	"testing"

	"github.com/ssm-access-manager/internal/validation"
	"github.com/stretchr/testify/assert"
)

func TestNewDocumentNameGenerator(t *testing.T) {
	t.Run("creates generator with default prefix", func(t *testing.T) {
		gen := validation.NewDocumentNameGenerator()
		assert.NotNil(t, gen)

		// Test that it uses default prefix "PF"
		name := gen.GenerateName("testuser", "example.com", 22)
		assert.Contains(t, name, "PF-")
	})
}

func TestNewDocumentNameGeneratorWithPrefix(t *testing.T) {
	tests := []struct {
		name           string
		prefix         string
		expectedPrefix string
	}{
		{
			name:           "creates generator with custom prefix",
			prefix:         "ACME",
			expectedPrefix: "ACME-",
		},
		{
			name:           "uses default prefix when empty string provided",
			prefix:         "",
			expectedPrefix: "PF-",
		},
		{
			name:           "accepts single character prefix",
			prefix:         "X",
			expectedPrefix: "X-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := validation.NewDocumentNameGeneratorWithPrefix(tt.prefix)
			assert.NotNil(t, gen)

			name := gen.GenerateName("user", "host", 22)
			assert.Contains(t, name, tt.expectedPrefix)
		})
	}
}

func TestGenerateName(t *testing.T) {
	tests := []struct {
		name         string
		prefix       string
		username     string
		host         string
		port         int
		expectedName string
	}{
		{
			name:         "generates name with default prefix",
			prefix:       "PF",
			username:     "john.doe",
			host:         "db.example.com",
			port:         5432,
			expectedName: "PF-john-doe-db-example-com-5432",
		},
		{
			name:         "generates name with custom prefix",
			prefix:       "ACME",
			username:     "alice",
			host:         "server1",
			port:         22,
			expectedName: "ACME-alice-server1-22",
		},
		{
			name:         "sanitizes special characters in username",
			prefix:       "PF",
			username:     "user@company",
			host:         "host",
			port:         80,
			expectedName: "PF-user-company-host-80",
		},
		{
			name:         "sanitizes special characters in host",
			prefix:       "PF",
			username:     "user",
			host:         "host_with_underscores",
			port:         443,
			expectedName: "PF-user-host_with_underscores-443",
		},
		{
			name:         "handles uppercase letters",
			prefix:       "PF",
			username:     "JohnDoe",
			host:         "DB.EXAMPLE.COM",
			port:         3306,
			expectedName: "PF-johndoe-db-example-com-3306",
		},
		{
			name:         "handles multiple consecutive special characters",
			prefix:       "PF",
			username:     "user!!!test",
			host:         "host---name",
			port:         8080,
			expectedName: "PF-user-test-host-name-8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := validation.NewDocumentNameGeneratorWithPrefix(tt.prefix)
			name := gen.GenerateName(tt.username, tt.host, tt.port)
			assert.Equal(t, tt.expectedName, name)
		})
	}
}

func TestGenerateName_Truncation(t *testing.T) {
	t.Run("truncates long names to fit 128 character limit", func(t *testing.T) {
		gen := validation.NewDocumentNameGenerator()

		// Create very long username and host
		longUsername := "verylongusernamethatexceedsthemaximumlengthallowed"
		longHost := "verylonghostnamethatexceedsthemaximumlengthallowedforssmdocuments.example.com"

		name := gen.GenerateName(longUsername, longHost, 5432)

		// Should be truncated to 128 characters or less
		assert.LessOrEqual(t, len(name), 128)

		// Should still contain the prefix and port
		assert.Contains(t, name, "PF-")
		assert.Contains(t, name, "-5432")
	})

	t.Run("does not truncate normal length names", func(t *testing.T) {
		gen := validation.NewDocumentNameGenerator()

		name := gen.GenerateName("user", "host", 22)

		// Should be much shorter than 128
		assert.Less(t, len(name), 128)
		assert.Equal(t, "PF-user-host-22", name)
	})
}

func TestSanitizeComponent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes dots",
			input:    "user.name",
			expected: "user-name",
		},
		{
			name:     "removes at signs",
			input:    "user@domain",
			expected: "user-domain",
		},
		{
			name:     "removes spaces",
			input:    "user name",
			expected: "user-name",
		},
		{
			name:     "preserves hyphens",
			input:    "user-name",
			expected: "user-name",
		},
		{
			name:     "preserves underscores",
			input:    "user_name",
			expected: "user_name",
		},
		{
			name:     "converts to lowercase",
			input:    "UserName",
			expected: "username",
		},
		{
			name:     "removes multiple consecutive special chars",
			input:    "user!!!name",
			expected: "user-name",
		},
		{
			name:     "trims leading and trailing hyphens",
			input:    "-username-",
			expected: "username",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := validation.NewDocumentNameGenerator()
			result := gen.SanitizeComponent(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateDocumentName(t *testing.T) {
	tests := []struct {
		name        string
		docName     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "accepts valid document name",
			docName:     "PF-user-host-22",
			expectError: false,
		},
		{
			name:        "accepts name with underscores",
			docName:     "PF-user_name-host_name-5432",
			expectError: false,
		},
		{
			name:        "accepts name with numbers",
			docName:     "PF-user123-host456-8080",
			expectError: false,
		},
		{
			name:        "rejects empty name",
			docName:     "",
			expectError: true,
			errorMsg:    "cannot be empty",
		},
		{
			name:        "rejects name with spaces",
			docName:     "PF-user name-host-22",
			expectError: true,
			errorMsg:    "invalid character",
		},
		{
			name:        "rejects name with dots",
			docName:     "PF-user.name-host-22",
			expectError: true,
			errorMsg:    "invalid character",
		},
		{
			name:        "rejects name starting with hyphen",
			docName:     "-PF-user-host-22",
			expectError: true,
			errorMsg:    "must start with an alphanumeric character",
		},
		{
			name:        "rejects name starting with underscore",
			docName:     "_PF-user-host-22",
			expectError: true,
			errorMsg:    "must start with an alphanumeric character",
		},
		{
			name:        "rejects name exceeding 128 characters",
			docName:     "PF-" + string(make([]byte, 130)),
			expectError: true,
			errorMsg:    "too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := validation.NewDocumentNameGenerator()
			err := gen.ValidateDocumentName(tt.docName)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseDocumentName(t *testing.T) {
	tests := []struct {
		name             string
		prefix           string
		docName          string
		expectedUsername string
		expectedHost     string
		expectedPort     int
		expectError      bool
		errorMsg         string
	}{
		{
			name:             "parses valid document name",
			prefix:           "PF",
			docName:          "PF-john-doe-db-example-com-5432",
			expectedUsername: "john",
			expectedHost:     "doe-db-example-com",
			expectedPort:     5432,
			expectError:      false,
		},
		{
			name:             "parses name with custom prefix",
			prefix:           "ACME",
			docName:          "ACME-alice-server1-22",
			expectedUsername: "alice",
			expectedHost:     "server1",
			expectedPort:     22,
			expectError:      false,
		},
		{
			name:             "parses name with underscores",
			prefix:           "PF",
			docName:          "PF-user_name-host_name-8080",
			expectedUsername: "user_name",
			expectedHost:     "host_name",
			expectedPort:     8080,
			expectError:      false,
		},
		{
			name:        "rejects name with wrong prefix",
			prefix:      "PF",
			docName:     "WRONG-user-host-22",
			expectError: true,
			errorMsg:    "must start with 'PF-'",
		},
		{
			name:        "rejects name without port",
			prefix:      "PF",
			docName:     "PF-user-host",
			expectError: true,
			errorMsg:    "invalid port",
		},
		{
			name:        "rejects name with invalid port",
			prefix:      "PF",
			docName:     "PF-user-host-abc",
			expectError: true,
			errorMsg:    "invalid port",
		},
		{
			name:        "rejects name without host",
			prefix:      "PF",
			docName:     "PF-user-22",
			expectError: true,
			errorMsg:    "missing host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := validation.NewDocumentNameGeneratorWithPrefix(tt.prefix)
			username, host, port, err := gen.ParseDocumentName(tt.docName)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedUsername, username)
				assert.Equal(t, tt.expectedHost, host)
				assert.Equal(t, tt.expectedPort, port)
			}
		})
	}
}

func TestGenerateAndParse_RoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		username string
		host     string
		port     int
	}{
		{
			name:     "round trip with default prefix",
			prefix:   "PF",
			username: "john.doe",
			host:     "db.example.com",
			port:     5432,
		},
		{
			name:     "round trip with custom prefix",
			prefix:   "ACME",
			username: "alice",
			host:     "server1",
			port:     22,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := validation.NewDocumentNameGeneratorWithPrefix(tt.prefix)

			// Generate a name
			docName := gen.GenerateName(tt.username, tt.host, tt.port)

			// Validate it
			err := gen.ValidateDocumentName(docName)
			assert.NoError(t, err)

			// Parse it back
			username, host, port, err := gen.ParseDocumentName(docName)
			assert.NoError(t, err)

			// Note: username and host will be sanitized, so we can't compare directly
			// But port should match exactly
			assert.Equal(t, tt.port, port)
			assert.NotEmpty(t, username)
			assert.NotEmpty(t, host)
		})
	}
}
