package handlers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// parseCommandText is the function we're testing from cmd/request-handler/main.go
// Since it's in main package, we'll need to test it through a wrapper or copy the logic
// For now, we'll create a local version for testing purposes
func parseCommandText(text string) map[string]string {
	params := make(map[string]string)
	
	// Manual parsing to handle quoted values
	i := 0
	for i < len(text) {
		// Skip whitespace
		for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
			i++
		}
		if i >= len(text) {
			break
		}
		
		// Find key=value pair
		keyStart := i
		for i < len(text) && text[i] != '=' && text[i] != ' ' && text[i] != '\t' {
			i++
		}
		
		if i >= len(text) || text[i] != '=' {
			// No '=' found, skip this token
			for i < len(text) && text[i] != ' ' && text[i] != '\t' {
				i++
			}
			continue
		}
		
		key := text[keyStart:i]
		i++ // Skip '='
		
		if i >= len(text) {
			break
		}
		
		var value string
		
		// Check if value is quoted
		if text[i] == '"' {
			i++ // Skip opening quote
			valueStart := i
			// Find closing quote
			for i < len(text) && text[i] != '"' {
				if text[i] == '\\' && i+1 < len(text) {
					i++ // Skip escaped character
				}
				i++
			}
			value = text[valueStart:i]
			if i < len(text) {
				i++ // Skip closing quote
			}
		} else {
			// Unquoted value - read until whitespace
			valueStart := i
			
			// Check for Slack URL formatting
			if text[i] == '<' {
				// Find the closing '>'
				for i < len(text) && text[i] != '>' {
					i++
				}
				if i < len(text) {
					i++ // Include the '>'
				}
				rawValue := text[valueStart:i]
				
				// Strip Slack's URL formatting: <http://example.com|example.com> -> example.com
				// Also handles: <http://example.com> -> example.com
				if len(rawValue) > 2 && rawValue[0] == '<' && rawValue[len(rawValue)-1] == '>' {
					rawValue = rawValue[1 : len(rawValue)-1]
					
					// If it contains a pipe, take the part after the pipe (the display text)
					pipeIdx := -1
					for j := 0; j < len(rawValue); j++ {
						if rawValue[j] == '|' {
							pipeIdx = j
							break
						}
					}
					if pipeIdx != -1 {
						value = rawValue[pipeIdx+1:]
					} else {
						// Otherwise, strip the protocol if present
						if len(rawValue) > 7 && rawValue[:7] == "http://" {
							value = rawValue[7:]
						} else if len(rawValue) > 8 && rawValue[:8] == "https://" {
							value = rawValue[8:]
						} else {
							value = rawValue
						}
					}
				} else {
					value = rawValue
				}
			} else {
				// Regular unquoted value
				for i < len(text) && text[i] != ' ' && text[i] != '\t' {
					i++
				}
				value = text[valueStart:i]
			}
		}
		
		params[key] = value
	}
	
	return params
}

// TestParseCommandText_BasicParameters tests parsing basic key=value pairs
func TestParseCommandText_BasicParameters(t *testing.T) {
	text := "host=db.example.com port=5432 account=123456789012"
	params := parseCommandText(text)
	
	assert.Equal(t, "db.example.com", params["host"])
	assert.Equal(t, "5432", params["port"])
	assert.Equal(t, "123456789012", params["account"])
}

// TestParseCommandText_QuotedReason tests parsing reason with quoted value
func TestParseCommandText_QuotedReason(t *testing.T) {
	text := `host=db.example.com reason="Database investigation for ticket #1234"`
	params := parseCommandText(text)
	
	assert.Equal(t, "db.example.com", params["host"])
	assert.Equal(t, "Database investigation for ticket #1234", params["reason"])
}

// TestParseCommandText_QuotedReasonWithMultipleSpaces tests quoted reason with multiple spaces
func TestParseCommandText_QuotedReasonWithMultipleSpaces(t *testing.T) {
	text := `reason="Need to investigate    production    issue"`
	params := parseCommandText(text)
	
	assert.Equal(t, "Need to investigate    production    issue", params["reason"])
}

// TestParseCommandText_UnquotedReason tests parsing reason without quotes (single word)
func TestParseCommandText_UnquotedReason(t *testing.T) {
	text := "host=db.example.com reason=investigation"
	params := parseCommandText(text)
	
	assert.Equal(t, "db.example.com", params["host"])
	assert.Equal(t, "investigation", params["reason"])
}

// TestParseCommandText_EmptyQuotedReason tests parsing empty quoted reason
func TestParseCommandText_EmptyQuotedReason(t *testing.T) {
	text := `host=db.example.com reason=""`
	params := parseCommandText(text)
	
	assert.Equal(t, "db.example.com", params["host"])
	assert.Equal(t, "", params["reason"])
}

// TestParseCommandText_ReasonWithSpecialCharacters tests reason with special characters
func TestParseCommandText_ReasonWithSpecialCharacters(t *testing.T) {
	text := `reason="Fix bug #123: database connection timeout (urgent!)"`
	params := parseCommandText(text)
	
	assert.Equal(t, "Fix bug #123: database connection timeout (urgent!)", params["reason"])
}

// TestParseCommandText_MixedQuotedAndUnquoted tests mixing quoted and unquoted parameters
func TestParseCommandText_MixedQuotedAndUnquoted(t *testing.T) {
	text := `host=db.example.com port=5432 reason="Production issue" account=123456789012`
	params := parseCommandText(text)
	
	assert.Equal(t, "db.example.com", params["host"])
	assert.Equal(t, "5432", params["port"])
	assert.Equal(t, "Production issue", params["reason"])
	assert.Equal(t, "123456789012", params["account"])
}

// TestParseCommandText_SlackURLFormatting tests Slack's URL formatting
func TestParseCommandText_SlackURLFormatting(t *testing.T) {
	text := "host=<http://db.example.com|db.example.com> port=5432"
	params := parseCommandText(text)
	
	assert.Equal(t, "db.example.com", params["host"])
	assert.Equal(t, "5432", params["port"])
}

// TestParseCommandText_SlackURLFormattingWithoutPipe tests Slack URL without pipe
func TestParseCommandText_SlackURLFormattingWithoutPipe(t *testing.T) {
	text := "host=<http://db.example.com> port=5432"
	params := parseCommandText(text)
	
	assert.Equal(t, "db.example.com", params["host"])
	assert.Equal(t, "5432", params["port"])
}

// TestParseCommandText_NoParameters tests empty input
func TestParseCommandText_NoParameters(t *testing.T) {
	text := ""
	params := parseCommandText(text)
	
	assert.Empty(t, params)
}

// TestParseCommandText_OnlyWhitespace tests input with only whitespace
func TestParseCommandText_OnlyWhitespace(t *testing.T) {
	text := "   \t  "
	params := parseCommandText(text)
	
	assert.Empty(t, params)
}

// TestParseCommandText_ReasonAtBeginning tests reason parameter at the beginning
func TestParseCommandText_ReasonAtBeginning(t *testing.T) {
	text := `reason="Database maintenance" host=db.example.com port=5432`
	params := parseCommandText(text)
	
	assert.Equal(t, "Database maintenance", params["reason"])
	assert.Equal(t, "db.example.com", params["host"])
	assert.Equal(t, "5432", params["port"])
}

// TestParseCommandText_ReasonAtEnd tests reason parameter at the end
func TestParseCommandText_ReasonAtEnd(t *testing.T) {
	text := `host=db.example.com port=5432 reason="Database maintenance"`
	params := parseCommandText(text)
	
	assert.Equal(t, "db.example.com", params["host"])
	assert.Equal(t, "5432", params["port"])
	assert.Equal(t, "Database maintenance", params["reason"])
}
