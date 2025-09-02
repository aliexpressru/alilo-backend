package util

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// TruncateString safety truncate string, even maxLen more that text length
func TruncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	// Convert to runes for proper Unicode handling
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}

	return string(runes[:maxLen])
}

func RandomMixedCaseString(size int) (string, error) {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	result := make([]rune, size)

	for i := range result {
		// Generate a cryptographically secure random index
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random string: %w", err)
		}
		result[i] = letters[n.Int64()]
	}

	return string(result), nil
}
