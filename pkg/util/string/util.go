package string

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// var logger = zap.S()

// MaskString Маскировка строки для безопасности
func MaskString(src string) string {
	if len(src) < 5 {
		return strings.Repeat("*", len(src))
	}

	return fmt.Sprint(src[:2], strings.Repeat("*", len(src)-4), src[len(src)-2:])
}

func ReplaceAllUnnecessarySymbols(original string) string {
	original = strings.ReplaceAll(original, "§", "_")
	original = strings.ReplaceAll(original, "±", "_")
	original = strings.ReplaceAll(original, "%", "p")
	re := regexp.MustCompile(`[[:punct:]]|[[:space:]]`)
	original = re.ReplaceAllString(original, "_")

	return original
}

func GetLastRune(s string, c int) string {
	j := len(s)
	for i := 0; i < c && j > 0; i++ {
		_, size := utf8.DecodeLastRuneInString(s[:j])
		j -= size
	}

	return s[j:]
}
