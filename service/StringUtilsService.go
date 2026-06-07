package service

import (
	"strings"
	"unicode"
)

// normalizeAndCompare normalizes both strings and compares them
func NormalizeAndCompare(str1, str2 string) bool {
	normalizeString := func(s string) string {
		var b strings.Builder
		b.Grow(len(s))
		lastWasSpace := false

		for _, r := range s {
			isWord := unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
			isSpace := unicode.IsSpace(r)

			if !isWord && !isSpace {
				continue
			}

			if isSpace {
				if !lastWasSpace && b.Len() > 0 {
					b.WriteByte(' ')
					lastWasSpace = true
				}
				continue
			}

			b.WriteRune(unicode.ToLower(r))
			lastWasSpace = false
		}

		res := b.String()
		if len(res) > 0 && res[len(res)-1] == ' ' {
			return res[:len(res)-1]
		}
		return res
	}

	// Normalize both strings
	normalizedStr1 := normalizeString(str1)
	normalizedStr2 := normalizeString(str2)

	// Compare the normalized strings
	return normalizedStr1 == normalizedStr2
}
