package shortcode

import (
	"crypto/sha256"
	"fmt"

	"github.com/akamensky/base58"
)

type ShortCode struct {
	UserID int64
	URL    string
	Length int
}

const base58Chars = "123456789abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ"

func New(userID int32, url string, length int) string {
	return GenerateShortCode(userID, url, length)
}

// Hash the URL and user ID to create a unique short code
func GenerateShortCode(userID int32, url string, length int) string {
	data := fmt.Sprintf("%d-%s", userID, url)

	hash := sha256.New()
	hash.Write([]byte(data))
	hashed := hash.Sum(nil)

	hashPart := hashed[:8]

	shortCode := base58.Encode(hashPart)

	if len(shortCode) > length {
		return shortCode[:length]
	}
	for len(shortCode) < length {
		shortCode += string(base58Chars[0])
	}
	return shortCode
}
