package service

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// normalizeAndCompare normalizes both strings and compares them
func NormalizeAndCompare(str1, str2 string) bool {
	normalizeString := func(s string) string {
		var b strings.Builder
		b.Grow(len(s))
		lastWasSpace := false

		for i := 0; i < len(s); {
			c := s[i]
			if c < utf8.RuneSelf {
				i++
				isSpace := c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f'
				isWord := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_'

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

				if c >= 'A' && c <= 'Z' {
					c += 32
				}
				b.WriteByte(c)
				lastWasSpace = false
			} else {
				r, size := utf8.DecodeRuneInString(s[i:])
				i += size

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
