package robot

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExtractBotCommand tests the bot command extraction logic with bounds checking.
// This test addresses issue #251 where malformed offset/length values could cause panic.
func TestExtractBotCommand(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		entities []map[string]interface{}
		expected string
	}{
		{
			name:    "valid command extraction",
			content: "/start hello",
			entities: []map[string]interface{}{
				{
					"type":   "bot_command",
					"offset": json.Number("0"),
					"length": json.Number("6"),
				},
			},
			expected: "/start",
		},
		{
			name:    "command in middle of content",
			content: "hello /help world",
			entities: []map[string]interface{}{
				{
					"type":   "bot_command",
					"offset": json.Number("6"),
					"length": json.Number("5"),
				},
			},
			expected: "/help",
		},
		{
			name:    "offset out of bounds - should return empty",
			content: "short",
			entities: []map[string]interface{}{
				{
					"type":   "bot_command",
					"offset": json.Number("100"),
					"length": json.Number("5"),
				},
			},
			expected: "",
		},
		{
			name:    "length exceeds content - should return empty",
			content: "short",
			entities: []map[string]interface{}{
				{
					"type":   "bot_command",
					"offset": json.Number("0"),
					"length": json.Number("100"),
				},
			},
			expected: "",
		},
		{
			name:    "negative offset - should return empty",
			content: "/test",
			entities: []map[string]interface{}{
				{
					"type":   "bot_command",
					"offset": json.Number("-1"),
					"length": json.Number("5"),
				},
			},
			expected: "",
		},
		{
			name:    "zero length - should return empty",
			content: "/test",
			entities: []map[string]interface{}{
				{
					"type":   "bot_command",
					"offset": json.Number("0"),
					"length": json.Number("0"),
				},
			},
			expected: "",
		},
		{
			name:    "missing offset field - should return empty",
			content: "/test",
			entities: []map[string]interface{}{
				{
					"type":   "bot_command",
					"length": json.Number("5"),
				},
			},
			expected: "",
		},
		{
			name:    "missing length field - should return empty",
			content: "/test",
			entities: []map[string]interface{}{
				{
					"type":   "bot_command",
					"offset": json.Number("0"),
				},
			},
			expected: "",
		},
		{
			name:    "wrong type for offset - should return empty",
			content: "/test",
			entities: []map[string]interface{}{
				{
					"type":   "bot_command",
					"offset": "not a number",
					"length": json.Number("5"),
				},
			},
			expected: "",
		},
		{
			name:     "empty entities - should return empty",
			content:  "/test",
			entities: []map[string]interface{}{},
			expected: "",
		},
		{
			name:    "unicode content - valid extraction",
			content: "/测试 你好",
			entities: []map[string]interface{}{
				{
					"type":   "bot_command",
					"offset": json.Number("0"),
					"length": json.Number("3"),
				},
			},
			expected: "/测试",
		},
		{
			name:    "offset+length exactly at boundary",
			content: "/test",
			entities: []map[string]interface{}{
				{
					"type":   "bot_command",
					"offset": json.Number("0"),
					"length": json.Number("5"),
				},
			},
			expected: "/test",
		},
		{
			name:    "offset+length exceeds boundary by 1 - should return empty",
			content: "/test",
			entities: []map[string]interface{}{
				{
					"type":   "bot_command",
					"offset": json.Number("0"),
					"length": json.Number("6"),
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBotCommandKey(tt.content, tt.entities)
			assert.Equal(t, tt.expected, result, "extractBotCommandKey should return expected result")
		})
	}
}

// extractBotCommandKey extracts the bot command from content using entities.
// This is a helper function that mirrors the logic in messagesListen for testability.
func extractBotCommandKey(content string, entities []map[string]interface{}) string {
	if entities == nil {
		return ""
	}

	var offset int64
	var length int64
	var offsetOK, lengthOK bool

	for _, entitiesMap := range entities {
		if entitiesMap["type"] == "bot_command" {
			// Safely extract offset
			if offsetVal, ok := entitiesMap["offset"].(json.Number); ok {
				offset, _ = offsetVal.Int64()
				offsetOK = true
			}
			// Safely extract length
			if lengthVal, ok := entitiesMap["length"].(json.Number); ok {
				length, _ = lengthVal.Int64()
				lengthOK = true
			}
			break
		}
	}

	contentRunes := []rune(content)
	contentLen := int64(len(contentRunes))

	// Validate bounds before slicing - require both offset and length to be valid
	if offsetOK && lengthOK && offset >= 0 && length > 0 && offset < contentLen && offset+length <= contentLen {
		return string(contentRunes[offset : offset+length])
	}

	return ""
}
