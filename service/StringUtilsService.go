package service

import (
	"strings"
	"unicode"
)

// NormalizeAndCompare normalizes both strings and compares them
func NormalizeAndCompare(str1, str2 string) bool {
	normalizeString := func(s string) string {
		var b strings.Builder
		b.Grow(len(s))
		lastSpace := true // Start as true to trim leading spaces
		for _, r := range s {
			r = unicode.ToLower(r)
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				b.WriteRune(r)
				lastSpace = false
			} else if unicode.IsSpace(r) {
				if !lastSpace {
					b.WriteByte(' ')
					lastSpace = true
				}
			}
		}
		res := b.String()
		if len(res) > 0 && res[len(res)-1] == ' ' {
			res = res[:len(res)-1]
		}
		return res
	}

	// Compare the normalized strings
	return normalizeString(str1) == normalizeString(str2)
}
