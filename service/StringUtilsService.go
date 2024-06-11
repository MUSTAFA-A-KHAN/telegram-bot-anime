package service

import (
	"regexp"
	"strings"
)

// normalizeAndCompare normalizes both strings and compares them
func NormalizeAndCompare(str1, str2 string) bool {
	normalizeString := func(s string) string {
		// Convert to lowercase
		s = strings.ToLower(s)
		// Remove punctuation using regex
		re := regexp.MustCompile(`[^\w\s]`)
		s = re.ReplaceAllString(s, "")
		// Remove extra whitespace
		s = strings.Join(strings.Fields(s), " ")
		return s
	}

	// Normalize both strings
	normalizedStr1 := normalizeString(str1)
	normalizedStr2 := normalizeString(str2)

	// Compare the normalized strings
	return normalizedStr1 == normalizedStr2
}
