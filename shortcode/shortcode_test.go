package shortcode

import (
	"testing"
)

func TestShortCodeLength(t *testing.T) {
	tests := []struct {
		userID     int32
		url        string
		length     int
		wantLength int
	}{
		{userID: 1, url: "https://example.com", length: 6, wantLength: 6},
		{userID: 2, url: "https://example.com", length: 8, wantLength: 8},
		{userID: 1, url: "https://example.com", length: 10, wantLength: 10},
		{userID: 1, url: "https://anotherexample.com", length: 8, wantLength: 8},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := GenerateShortCode(tt.userID, tt.url, tt.length)
			gotLength := len(got)
			if gotLength != tt.wantLength {
				t.Errorf("Expected %v, got %v", tt.wantLength, gotLength)
			}
		})
	}
}

func TestUniqueForSameURL(t *testing.T) {
	// Same userID, same URL should generate unique short code
	url := "https://example.com"
	userID := int32(1)
	length := 7

	code1 := GenerateShortCode(userID, url, length)
	code2 := GenerateShortCode(userID, url, length)

	if code1 == code2 {
		t.Errorf("Expected different short codes, got same: %v", code1)
	}
}

func TestUniqueToUser(t *testing.T) {
	// Different userIDs, same URL should generate unique short codes
	url := "https://example.com"
	userID1 := int32(1)
	userID2 := int32(2)
	length := 8

	code1 := GenerateShortCode(userID1, url, length)
	code2 := GenerateShortCode(userID2, url, length)

	if code1 == code2 {
		t.Errorf("Expected different short codes, got same: %v", code1)
	}
}
