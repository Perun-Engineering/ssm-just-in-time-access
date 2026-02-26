package validation

import (
	"fmt"
	"strings"
)

// DocumentNameGenerator generates standardized SSM document names
type DocumentNameGenerator struct {
	validator *RequestValidator
	prefix    string
}

// NewDocumentNameGenerator creates a new document name generator with default prefix
func NewDocumentNameGenerator() *DocumentNameGenerator {
	return NewDocumentNameGeneratorWithPrefix("PF")
}

// NewDocumentNameGeneratorWithPrefix creates a new document name generator with custom prefix
func NewDocumentNameGeneratorWithPrefix(prefix string) *DocumentNameGenerator {
	if prefix == "" {
		prefix = "PF"
	}
	return &DocumentNameGenerator{
		validator: NewRequestValidator(90),
		prefix:    prefix,
	}
}

// GenerateName generates a document name in the format: {PREFIX}-{username}-{host}-{port}
// All components are sanitized to ensure AWS SSM naming compliance
func (g *DocumentNameGenerator) GenerateName(username, host string, port int) string {
	// Sanitize each component
	sanitizedUsername := g.SanitizeComponent(username)
	sanitizedHost := g.SanitizeComponent(host)
	
	// Build the document name
	documentName := fmt.Sprintf("%s-%s-%s-%d", g.prefix, sanitizedUsername, sanitizedHost, port)
	
	// Ensure the final name is valid (max 128 characters for SSM documents)
	if len(documentName) > 128 {
		// Truncate if necessary, keeping the format recognizable
		maxUsernameLen := 30
		maxHostLen := 60
		
		if len(sanitizedUsername) > maxUsernameLen {
			sanitizedUsername = sanitizedUsername[:maxUsernameLen]
		}
		if len(sanitizedHost) > maxHostLen {
			sanitizedHost = sanitizedHost[:maxHostLen]
		}
		
		documentName = fmt.Sprintf("%s-%s-%s-%d", g.prefix, sanitizedUsername, sanitizedHost, port)
	}
	
	return documentName
}

// SanitizeComponent sanitizes a string component for use in document names
func (g *DocumentNameGenerator) SanitizeComponent(component string) string {
	return g.validator.SanitizeForDocumentName(component)
}

// ValidateDocumentName validates that a document name meets AWS SSM requirements
func (g *DocumentNameGenerator) ValidateDocumentName(name string) error {
	if name == "" {
		return fmt.Errorf("document name cannot be empty")
	}
	
	// Check length (AWS SSM document names can be up to 128 characters)
	if len(name) > 128 {
		return fmt.Errorf("document name too long: maximum 128 characters, got %d", len(name))
	}
	
	// Check that it only contains valid characters (alphanumeric, hyphens, underscores)
	for _, char := range name {
		if !isValidDocumentNameChar(char) {
			return fmt.Errorf("document name contains invalid character: %c (only alphanumeric, hyphens, and underscores allowed)", char)
		}
	}
	
	// Must start with an alphanumeric character
	if len(name) > 0 && !isAlphanumeric(rune(name[0])) {
		return fmt.Errorf("document name must start with an alphanumeric character")
	}
	
	return nil
}

// isValidDocumentNameChar checks if a character is valid for SSM document names
func isValidDocumentNameChar(char rune) bool {
	return isAlphanumeric(char) || char == '-' || char == '_'
}

// isAlphanumeric checks if a character is alphanumeric
func isAlphanumeric(char rune) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')
}

// ParseDocumentName parses a document name and extracts username, host, and port
// Returns error if the name doesn't follow the expected format
func (g *DocumentNameGenerator) ParseDocumentName(name string) (username, host string, port int, err error) {
	// Expected format: {PREFIX}-{username}-{host}-{port}
	expectedPrefix := g.prefix + "-"
	if !strings.HasPrefix(name, expectedPrefix) {
		return "", "", 0, fmt.Errorf("document name must start with '%s'", expectedPrefix)
	}
	
	// Remove the prefix
	remainder := strings.TrimPrefix(name, expectedPrefix)
	
	// Split by the last hyphen to get the port
	lastHyphenIdx := strings.LastIndex(remainder, "-")
	if lastHyphenIdx == -1 {
		return "", "", 0, fmt.Errorf("invalid document name format: missing port")
	}
	
	portStr := remainder[lastHyphenIdx+1:]
	_, err = fmt.Sscanf(portStr, "%d", &port)
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid port in document name: %w", err)
	}
	
	// The remainder before the port contains username and host
	usernameAndHost := remainder[:lastHyphenIdx]
	
	// Find the second hyphen to separate username from host
	secondHyphenIdx := strings.Index(usernameAndHost, "-")
	if secondHyphenIdx == -1 {
		return "", "", 0, fmt.Errorf("invalid document name format: missing host")
	}
	
	username = usernameAndHost[:secondHyphenIdx]
	host = usernameAndHost[secondHyphenIdx+1:]
	
	return username, host, port, nil
}
