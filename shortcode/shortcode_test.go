package shortcode

import (
	"testing"
)

func TestGenerateShortCode(t *testing.T) {
	tests := []struct {
		name   string
		userID int32
		url    string
		length int
	}{
		{
			name:   "Consistent output for same input",
			userID: 12345,
			url:    "https://example.com",
			length: 6,
		},
		{
			name:   "Different URLs produce different codes",
			userID: 12345,
			url:    "https://example.com/page1",
			length: 7,
		},
		{
			name:   "Different user IDs produce different codes",
			userID: 54321,
			url:    "https://example.com",
			length: 8,
		},
		{
			name:   "Edge case with empty URL",
			userID: 0,
			url:    "",
			length: 6,
		},
	}

	shortCodes := make(map[string]bool) // To check uniqueness

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := GenerateShortCode(tt.userID, tt.url, tt.length)
			if code == "" {
				t.Errorf("Generated short code is empty")
			}

			if _, exists := shortCodes[code]; exists {
				t.Errorf("Duplicate short code generated for input: userID=%d, url=%s", tt.userID, tt.url)
			}
			shortCodes[code] = true

			t.Logf("Generated short code for userID=%d, url=%s: %s", tt.userID, tt.url, code)
		})
	}
}

func TestConsistency(t *testing.T) {
	userID := int32(12345)
	url := "https://example.com"

	code1 := GenerateShortCode(userID, url, 7)
	code2 := GenerateShortCode(userID, url, 7)

	if code1 != code2 {
		t.Errorf("Short code is not consistent. Expected %s, got %s", code1, code2)
	}
}
