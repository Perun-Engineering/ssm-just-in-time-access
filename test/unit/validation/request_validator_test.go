package validation

import (
	"strings"
	"testing"
	"time"

	"github.com/ssm-access-manager/internal/validation"
	"github.com/stretchr/testify/assert"
)

func TestNewRequestValidator(t *testing.T) {
	t.Run("creates validator with specified max days", func(t *testing.T) {
		validator := validation.NewRequestValidator(30)
		assert.NotNil(t, validator)
	})

	t.Run("uses default max days when zero provided", func(t *testing.T) {
		validator := validation.NewRequestValidator(0)
		assert.NotNil(t, validator)

		// Test that it uses default 90 days
		futureDate := time.Now().AddDate(0, 0, 91)
		result := validator.ValidateExpirationDate(futureDate)
		assert.False(t, result.IsValid)
	})

	t.Run("uses default max days when negative provided", func(t *testing.T) {
		validator := validation.NewRequestValidator(-10)
		assert.NotNil(t, validator)
	})
}

func TestValidateHost(t *testing.T) {
	validator := validation.NewRequestValidator(90)

	tests := []struct {
		name        string
		host        string
		expectValid bool
		errorMsg    string
	}{
		// Valid hostnames
		{
			name:        "accepts simple hostname",
			host:        "server1",
			expectValid: true,
		},
		{
			name:        "accepts FQDN",
			host:        "db.example.com",
			expectValid: true,
		},
		{
			name:        "accepts hostname with hyphens",
			host:        "web-server-01",
			expectValid: true,
		},
		{
			name:        "accepts subdomain",
			host:        "api.staging.example.com",
			expectValid: true,
		},
		{
			name:        "accepts hostname with numbers",
			host:        "server123",
			expectValid: true,
		},
		// Valid IP addresses
		{
			name:        "accepts IPv4 address",
			host:        "192.168.1.1",
			expectValid: true,
		},
		{
			name:        "accepts IPv6 address",
			host:        "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			expectValid: true,
		},
		{
			name:        "accepts IPv6 short form",
			host:        "2001:db8::1",
			expectValid: true,
		},
		// Invalid hostnames
		{
			name:        "rejects empty host",
			host:        "",
			expectValid: false,
			errorMsg:    "required",
		},
		{
			name:        "rejects host with spaces",
			host:        "host name",
			expectValid: false,
			errorMsg:    "invalid host format",
		},
		{
			name:        "rejects host starting with hyphen",
			host:        "-hostname",
			expectValid: false,
			errorMsg:    "invalid host format",
		},
		{
			name:        "rejects host ending with hyphen",
			host:        "hostname-",
			expectValid: false,
			errorMsg:    "invalid host format",
		},
		{
			name:        "rejects host with double dots",
			host:        "host..example.com",
			expectValid: false,
			errorMsg:    "invalid host format",
		},
		{
			name:        "rejects host starting with dot",
			host:        ".example.com",
			expectValid: false,
			errorMsg:    "invalid host format",
		},
		{
			name:        "rejects host ending with dot",
			host:        "example.com.",
			expectValid: false,
			errorMsg:    "invalid host format",
		},
		{
			name:        "rejects host with special characters",
			host:        "host@example.com",
			expectValid: false,
			errorMsg:    "invalid host format",
		},
		{
			name:        "rejects host with underscores",
			host:        "host_name",
			expectValid: false,
			errorMsg:    "invalid host format",
		},
		{
			name:        "rejects host exceeding 253 characters",
			host:        "verylonghostname" + strings.Repeat("a", 240) + ".com",
			expectValid: false,
			errorMsg:    "too long",
		},
		{
			name:        "rejects host with label exceeding 63 characters",
			host:        strings.Repeat("a", 64) + ".example.com",
			expectValid: false,
			errorMsg:    "label is too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateHost(tt.host)

			if tt.expectValid {
				assert.True(t, result.IsValid, "Expected host to be valid: %s", tt.host)
			} else {
				assert.False(t, result.IsValid, "Expected host to be invalid: %s", tt.host)
				assert.Contains(t, result.ErrorMessage, tt.errorMsg)
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	validator := validation.NewRequestValidator(90)

	tests := []struct {
		name        string
		port        int
		expectValid bool
		errorMsg    string
	}{
		// Valid ports
		{
			name:        "accepts port 1",
			port:        1,
			expectValid: true,
		},
		{
			name:        "accepts port 22 (SSH)",
			port:        22,
			expectValid: true,
		},
		{
			name:        "accepts port 80 (HTTP)",
			port:        80,
			expectValid: true,
		},
		{
			name:        "accepts port 443 (HTTPS)",
			port:        443,
			expectValid: true,
		},
		{
			name:        "accepts port 5432 (PostgreSQL)",
			port:        5432,
			expectValid: true,
		},
		{
			name:        "accepts port 8080",
			port:        8080,
			expectValid: true,
		},
		{
			name:        "accepts port 65535 (max)",
			port:        65535,
			expectValid: true,
		},
		// Invalid ports
		{
			name:        "rejects port 0",
			port:        0,
			expectValid: false,
			errorMsg:    "must be between 1 and 65535",
		},
		{
			name:        "rejects negative port",
			port:        -1,
			expectValid: false,
			errorMsg:    "must be between 1 and 65535",
		},
		{
			name:        "rejects port exceeding 65535",
			port:        65536,
			expectValid: false,
			errorMsg:    "must be between 1 and 65535",
		},
		{
			name:        "rejects very large port",
			port:        100000,
			expectValid: false,
			errorMsg:    "must be between 1 and 65535",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidatePort(tt.port)

			if tt.expectValid {
				assert.True(t, result.IsValid, "Expected port to be valid: %d", tt.port)
			} else {
				assert.False(t, result.IsValid, "Expected port to be invalid: %d", tt.port)
				assert.Contains(t, result.ErrorMessage, tt.errorMsg)
			}
		})
	}
}

func TestValidateExpirationDate(t *testing.T) {
	validator := validation.NewRequestValidator(90)
	now := time.Now()

	tests := []struct {
		name        string
		date        time.Time
		expectValid bool
		errorMsg    string
	}{
		// Valid dates
		{
			name:        "accepts date 1 day in future",
			date:        now.AddDate(0, 0, 1),
			expectValid: true,
		},
		{
			name:        "accepts date 14 days in future",
			date:        now.AddDate(0, 0, 14),
			expectValid: true,
		},
		{
			name:        "accepts date 30 days in future",
			date:        now.AddDate(0, 0, 30),
			expectValid: true,
		},
		{
			name:        "accepts date 90 days in future (max)",
			date:        now.AddDate(0, 0, 90),
			expectValid: true,
		},
		{
			name:        "accepts date 1 hour in future",
			date:        now.Add(1 * time.Hour),
			expectValid: true,
		},
		// Invalid dates
		{
			name:        "rejects date in the past",
			date:        now.AddDate(0, 0, -1),
			expectValid: false,
			errorMsg:    "must be in the future",
		},
		{
			name:        "rejects date 1 year ago",
			date:        now.AddDate(-1, 0, 0),
			expectValid: false,
			errorMsg:    "must be in the future",
		},
		{
			name:        "rejects date 91 days in future (exceeds max)",
			date:        now.AddDate(0, 0, 91),
			expectValid: false,
			errorMsg:    "must be within 90 days",
		},
		{
			name:        "rejects date 1 year in future",
			date:        now.AddDate(1, 0, 0),
			expectValid: false,
			errorMsg:    "must be within 90 days",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateExpirationDate(tt.date)

			if tt.expectValid {
				assert.True(t, result.IsValid, "Expected date to be valid")
			} else {
				assert.False(t, result.IsValid, "Expected date to be invalid")
				assert.Contains(t, result.ErrorMessage, tt.errorMsg)
			}
		})
	}
}

func TestValidateExpirationDate_CustomMaxDays(t *testing.T) {
	validator := validation.NewRequestValidator(30)
	now := time.Now()

	t.Run("accepts date within custom max days", func(t *testing.T) {
		date := now.AddDate(0, 0, 29)
		result := validator.ValidateExpirationDate(date)
		assert.True(t, result.IsValid)
	})

	t.Run("rejects date exceeding custom max days", func(t *testing.T) {
		date := now.AddDate(0, 0, 31)
		result := validator.ValidateExpirationDate(date)
		assert.False(t, result.IsValid)
		assert.Contains(t, result.ErrorMessage, "must be within 30 days")
	})
}

func TestValidateUsername(t *testing.T) {
	validator := validation.NewRequestValidator(90)

	tests := []struct {
		name        string
		username    string
		expectValid bool
		errorMsg    string
	}{
		// Valid usernames
		{
			name:        "accepts simple username",
			username:    "john",
			expectValid: true,
		},
		{
			name:        "accepts username with dot",
			username:    "john.doe",
			expectValid: true,
		},
		{
			name:        "accepts username with hyphen",
			username:    "john-doe",
			expectValid: true,
		},
		{
			name:        "accepts username with underscore",
			username:    "john_doe",
			expectValid: true,
		},
		{
			name:        "accepts username with numbers",
			username:    "user123",
			expectValid: true,
		},
		{
			name:        "accepts email-like username",
			username:    "john.doe@company",
			expectValid: true,
		},
		// Invalid usernames
		{
			name:        "rejects empty username",
			username:    "",
			expectValid: false,
			errorMsg:    "required",
		},
		{
			name:        "rejects whitespace-only username",
			username:    "   ",
			expectValid: false,
			errorMsg:    "cannot be empty or whitespace only",
		},
		{
			name:        "rejects username exceeding 64 characters",
			username:    string(make([]byte, 65)),
			expectValid: false,
			errorMsg:    "too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateUsername(tt.username)

			if tt.expectValid {
				assert.True(t, result.IsValid, "Expected username to be valid: %s", tt.username)
			} else {
				assert.False(t, result.IsValid, "Expected username to be invalid: %s", tt.username)
				assert.Contains(t, result.ErrorMessage, tt.errorMsg)
			}
		})
	}
}

func TestValidateAccountID(t *testing.T) {
	validator := validation.NewRequestValidator(90)

	tests := []struct {
		name        string
		accountID   string
		expectValid bool
		errorMsg    string
	}{
		// Valid account IDs
		{
			name:        "accepts valid 12-digit account ID",
			accountID:   "123456789012",
			expectValid: true,
		},
		{
			name:        "accepts account ID starting with zeros",
			accountID:   "000000000001",
			expectValid: true,
		},
		// Invalid account IDs
		{
			name:        "rejects empty account ID",
			accountID:   "",
			expectValid: false,
			errorMsg:    "required",
		},
		{
			name:        "rejects account ID with 11 digits",
			accountID:   "12345678901",
			expectValid: false,
			errorMsg:    "must be a 12-digit number",
		},
		{
			name:        "rejects account ID with 13 digits",
			accountID:   "1234567890123",
			expectValid: false,
			errorMsg:    "must be a 12-digit number",
		},
		{
			name:        "rejects account ID with letters",
			accountID:   "12345678901a",
			expectValid: false,
			errorMsg:    "must be a 12-digit number",
		},
		{
			name:        "rejects account ID with special characters",
			accountID:   "123456-78901",
			expectValid: false,
			errorMsg:    "must be a 12-digit number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateAccountID(tt.accountID)

			if tt.expectValid {
				assert.True(t, result.IsValid, "Expected account ID to be valid: %s", tt.accountID)
			} else {
				assert.False(t, result.IsValid, "Expected account ID to be invalid: %s", tt.accountID)
				assert.Contains(t, result.ErrorMessage, tt.errorMsg)
			}
		})
	}
}

func TestSanitizeForDocumentName(t *testing.T) {
	validator := validation.NewRequestValidator(90)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "preserves alphanumeric characters",
			input:    "user123",
			expected: "user123",
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
			name:     "replaces dots with hyphens",
			input:    "user.name",
			expected: "user-name",
		},
		{
			name:     "replaces at signs with hyphens",
			input:    "user@domain",
			expected: "user-domain",
		},
		{
			name:     "replaces spaces with hyphens",
			input:    "user name",
			expected: "user-name",
		},
		{
			name:     "replaces special characters with hyphens",
			input:    "user!@#$%name",
			expected: "user-name",
		},
		{
			name:     "collapses multiple hyphens",
			input:    "user---name",
			expected: "user-name",
		},
		{
			name:     "trims leading hyphens",
			input:    "---username",
			expected: "username",
		},
		{
			name:     "trims trailing hyphens",
			input:    "username---",
			expected: "username",
		},
		{
			name:     "converts to lowercase",
			input:    "UserName",
			expected: "username",
		},
		{
			name:     "handles mixed case and special chars",
			input:    "User.Name@Domain",
			expected: "user-name-domain",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "handles only special characters",
			input:    "!@#$%",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.SanitizeForDocumentName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
