package service

import (
	"strings"
)

// FontStyle represents different text formatting styles
type FontStyle struct {
	Name        string
	Description string
	Converter   func(string) string
}

// GetAllFontStyles returns all available font style converters
func GetAllFontStyles() []FontStyle {
	return []FontStyle{
		{Name: "Bold", Description: "Mathematical Bold", Converter: ToBold},
		{Name: "Italic", Description: "Mathematical Italic", Converter: ToItalic},
		{Name: "Bold Italic", Description: "Mathematical Bold Italic", Converter: ToBoldItalic},
		{Name: "Monospace", Description: "Monospace", Converter: ToMonospace},
		{Name: "Double Struck", Description: "Double Struck", Converter: ToDoubleStruck},
		{Name: "Sans Serif", Description: "Sans Serif", Converter: ToSansSerif},
		{Name: "Bold Sans", Description: "Bold Sans Serif", Converter: ToBoldSansSerif},
		{Name: "Italic Sans", Description: "Italic Sans Serif", Converter: ToItalicSansSerif},
		{Name: "Bold Italic Sans", Description: "Bold Italic Sans Serif", Converter: ToBoldItalicSansSerif},
		{Name: "Script", Description: "Script/Cursive", Converter: ToScript},
		{Name: "Bold Script", Description: "Bold Script", Converter: ToBoldScript},
		{Name: "Fraktur", Description: "Fraktur", Converter: ToFraktur},
		{Name: "Bold Fraktur", Description: "Bold Fraktur", Converter: ToBoldFraktur},
		{Name: "Small Caps", Description: "Small Caps", Converter: ToSmallCaps},
		{Name: "Reversed", Description: "Reversed Text", Converter: ToReversed},
		{Name: "Wide", Description: "Wide Text", Converter: ToWide},
	}
}

// ToBold converts text to Mathematical Bold
func ToBold(s string) string {
	result := strings.Builder{}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			result.WriteRune(r - 'a' + '𝐚')
		case r >= 'A' && r <= 'Z':
			result.WriteRune(r - 'A' + '𝐀')
		case r >= '0' && r <= '9':
			result.WriteRune(r - '0' + '𝟎')
		case r == ' ':
			result.WriteRune(' ')
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ToItalic converts text to Mathematical Italic
func ToItalic(s string) string {
	result := strings.Builder{}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			result.WriteRune(r - 'a' + '𝑎')
		case r >= 'A' && r <= 'Z':
			result.WriteRune(r - 'A' + '𝐴')
		case r == ' ':
			result.WriteRune(' ')
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ToBoldItalic converts text to Mathematical Bold Italic
func ToBoldItalic(s string) string {
	result := strings.Builder{}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			result.WriteRune(r - 'a' + '𝒂')
		case r >= 'A' && r <= 'Z':
			result.WriteRune(r - 'A' + '𝑨')
		case r == ' ':
			result.WriteRune(' ')
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ToMonospace converts text to Monospace
func ToMonospace(s string) string {
	result := strings.Builder{}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			result.WriteRune(r - 'a' + '𝚊')
		case r >= 'A' && r <= 'Z':
			result.WriteRune(r - 'A' + '𝙰')
		case r >= '0' && r <= '9':
			result.WriteRune(r - '0' + '𝟶')
		case r == ' ':
			result.WriteRune(' ')
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ToDoubleStruck converts text to Double Struck
func ToDoubleStruck(s string) string {
	result := strings.Builder{}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			result.WriteRune(r - 'a' + '𝕒')
		case r >= 'A' && r <= 'Z':
			result.WriteRune(r - 'A' + '𝔸')
		case r >= '0' && r <= '9':
			result.WriteRune(r - '0' + '𝟘')
		case r == ' ':
			result.WriteRune(' ')
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ToSansSerif converts text to Sans Serif
func ToSansSerif(s string) string {
	result := strings.Builder{}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			result.WriteRune(r - 'a' + '𝖺')
		case r >= 'A' && r <= 'Z':
			result.WriteRune(r - 'A' + '𝖠')
		case r >= '0' && r <= '9':
			result.WriteRune(r - '0' + '𝟢')
		case r == ' ':
			result.WriteRune(' ')
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ToBoldSansSerif converts text to Bold Sans Serif
func ToBoldSansSerif(s string) string {
	result := strings.Builder{}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			result.WriteRune(r - 'a' + '𝗮')
		case r >= 'A' && r <= 'Z':
			result.WriteRune(r - 'A' + '𝗔')
		case r >= '0' && r <= '9':
			result.WriteRune(r - '0' + '𝟬')
		case r == ' ':
			result.WriteRune(' ')
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ToItalicSansSerif converts text to Italic Sans Serif
func ToItalicSansSerif(s string) string {
	result := strings.Builder{}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			result.WriteRune(r - 'a' + '𝘢')
		case r >= 'A' && r <= 'Z':
			result.WriteRune(r - 'A' + '𝘈')
		case r == ' ':
			result.WriteRune(' ')
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ToBoldItalicSansSerif converts text to Bold Italic Sans Serif
func ToBoldItalicSansSerif(s string) string {
	result := strings.Builder{}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			result.WriteRune(r - 'a' + '𝙖')
		case r >= 'A' && r <= 'Z':
			result.WriteRune(r - 'A' + '𝘼')
		case r == ' ':
			result.WriteRune(' ')
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ToScript converts text to Script/Cursive
func ToScript(s string) string {
	result := strings.Builder{}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			result.WriteRune(r - 'a' + '𝒶')
		case r >= 'A' && r <= 'Z':
			result.WriteRune(r - 'A' + '𝒜')
		case r == ' ':
			result.WriteRune(' ')
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ToBoldScript converts text to Bold Script
func ToBoldScript(s string) string {
	result := strings.Builder{}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			result.WriteRune(r - 'a' + '𝓪')
		case r >= 'A' && r <= 'Z':
			result.WriteRune(r - 'A' + '𝓐')
		case r == ' ':
			result.WriteRune(' ')
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ToFraktur converts text to Fraktur
func ToFraktur(s string) string {
	result := strings.Builder{}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			result.WriteRune(r - 'a' + '𝔞')
		case r >= 'A' && r <= 'Z':
			result.WriteRune(r - 'A' + '𝔄')
		case r == ' ':
			result.WriteRune(' ')
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ToBoldFraktur converts text to Bold Fraktur
func ToBoldFraktur(s string) string {
	result := strings.Builder{}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			result.WriteRune(r - 'a' + '𝖆')
		case r >= 'A' && r <= 'Z':
			result.WriteRune(r - 'A' + '𝕬')
		case r == ' ':
			result.WriteRune(' ')
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ToSmallCaps converts text to Small Caps
func ToSmallCaps(s string) string {
	result := strings.Builder{}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			result.WriteRune(r - 'a' + 'ᴀ')
		case r >= 'A' && r <= 'Z':
			result.WriteRune(r - 'A' + 'ᴀ')
		case r >= '0' && r <= '9':
			result.WriteRune(r)
		case r == ' ':
			result.WriteRune(' ')
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ToReversed converts text to Reversed
func ToReversed(s string) string {
	result := strings.Builder{}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			result.WriteRune(r - 'a' + 'ɐ')
		case r >= 'A' && r <= 'Z':
			result.WriteRune(r - 'A' + 'Ɐ')
		case r == ' ':
			result.WriteRune(' ')
		default:
			result.WriteRune(r)
		}
	}
	// Reverse the entire string
	reversed := result.String()
	// Convert to rune slice for proper reversal
	runes := []rune(reversed)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// ToWide converts text to Wide/Fullwidth
func ToWide(s string) string {
	result := strings.Builder{}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			result.WriteRune(r - 'a' + 'ａ')
		case r >= 'A' && r <= 'Z':
			result.WriteRune(r - 'A' + 'Ａ')
		case r >= '0' && r <= '9':
			result.WriteRune(r - '0' + '０')
		case r == ' ':
			result.WriteRune('　')
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ConvertToStyle converts text to the specified style
func ConvertToStyle(text string, styleName string) string {
	styles := GetAllFontStyles()
	for _, style := range styles {
		if style.Name == styleName {
			return style.Converter(text)
		}
	}
	return text // Return original if style not found
}

// GetAllStylesForText returns a map of all styles applied to the text
func GetAllStylesForText(text string) map[string]string {
	styles := GetAllFontStyles()
	result := make(map[string]string)
	for _, style := range styles {
		result[style.Name] = style.Converter(text)
	}
	return result
}
