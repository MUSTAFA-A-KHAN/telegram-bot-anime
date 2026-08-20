package modbot

import (
	"strings"
)

// ArabicNormalize normalizes Arabic text by:
// 1. Removing tatweel/kashida (ـ)
// 2. Removing Arabic diacritics (tashkeel: فتحة, ضمة, كسرة, سكون, etc.)
// 3. Removing zero-width characters (ZWJ, ZWNJ, LRM, RLM)
// 4. Normalizing alef forms (أ إ آ -> ا)
// 5. Normalizing yaa/alf maqsura (ى -> ي)
// 6. Normalizing taa marbouta (ة -> ه)
// 7. Collapsing multiple spaces
func ArabicNormalize(input string) string {
	if input == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(input))

	normalized := normalizeArabicChars(input)

	for _, r := range normalized {
		switch {
		case r == tatweel || r == zwj || r == zwnj || r == lrm || r == rlm:
			continue // Skip formatting/control characters
		case isArabicTashkeel(r):
			continue // Skip diacritics (tashkeel)
		default:
			b.WriteRune(r)
		}
	}

	return strings.TrimSpace(collapseSpaces(b.String()))
}

func normalizeRuleTrigger(input string) string {
	normalized := strings.ToLower(strings.TrimSpace(input))
	normalized = ArabicNormalize(normalized)
	normalized = strings.TrimSpace(normalized)

	words := strings.Fields(normalized)
	for i, word := range words {
		for strings.HasPrefix(word, "ال") {
			word = strings.TrimPrefix(word, "ال")
		}
		words[i] = word
	}

	return strings.Join(words, " ")
}

func findRuleKeyByNormalizedTrigger(settings *ModChatSettings, trigger string) (string, bool) {
	normalizedTrigger := normalizeRuleTrigger(trigger)
	for key := range settings.Rules {
		if normalizeRuleTrigger(key) == normalizedTrigger {
			return key, true
		}
	}
	return "", false
}

// Arabic normalization constants
const (
	tatweel = '\u0640' // Arabic tatweel/kashida (ـ)
	zwj     = '\u200D' // Zero-width joiner
	zwnj    = '\u200C' // Zero-width non-joiner
	lrm     = '\u200E' // Left-to-right mark
	rlm     = '\u200F' // Right-to-left mark
)

// Arabic script ranges
const (
	arabicBlockStart       = '\u0600'
	arabicBlockEnd         = '\u06FF'
	arabicSupplement       = '\u0750'
	arabicSupplementEnd    = '\u077F'
	arabicExtendedAStart   = '\u08A0'
	arabicExtendedAEnd     = '\u08FF'
	arabicPresentationA    = '\uFB50'
	arabicPresentationAEnd = '\uFDFF'
	arabicPresentationB    = '\uFE70'
	arabicPresentationBEnd = '\uFEFF'
)

// isArabicTashkeel returns true if the rune is an Arabic diacritical mark.
func isArabicTashkeel(r rune) bool {
	return (r >= '\u064B' && r <= '\u065F') ||
		(r >= '\u0610' && r <= '\u061A') ||
		r == '\u06D6' || r == '\u06D7' || r == '\u06D8' || r == '\u06D9' ||
		r == '\u06DA' || r == '\u06DB' || r == '\u06DC' || r == '\u06DF' ||
		r == '\u06E0' || r == '\u06E1' || r == '\u06E2' || r == '\u06E3' ||
		r == '\u06E4' || r == '\u06E5' || r == '\u06E6' || r == '\u06E7' ||
		r == '\u06E8' || r == '\u06EA' || r == '\u06EB' || r == '\u06EC' ||
		r == '\u06ED'
}

// normalizeArabicChars normalizes variant letter forms to their base forms.
func normalizeArabicChars(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	for _, r := range s {
		switch {
		case r == '\u0622' || r == '\u0623' || r == '\u0625': // آ أ إ -> ا
			b.WriteRune('\u0627')
		case r == '\u0649': // ى (alif maqsura) -> ي
			b.WriteRune('\u064A')
		case r == '\u0629': // ة (taa marbouta) -> ه
			b.WriteRune('\u0647')
		case r >= arabicPresentationA && r <= arabicPresentationAEnd:
			b.WriteRune(mapArabicPresentationA(r))
		case r >= arabicPresentationB && r <= arabicPresentationBEnd:
			b.WriteRune(mapArabicPresentationB(r))
		default:
			b.WriteRune(r)
		}
	}

	return b.String()
}
