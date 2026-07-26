package modbot

import (
        "strings"
)

// mapArabicPresentationA maps Arabic Presentation Forms-A (FB50-FDFF) to base forms.
// These include ligatures and positional letter forms.
func mapArabicPresentationA(r rune) rune {
        switch r {
        case '\uFDF2': // ﷲ (Allah ligature)
                return '\u0627' // ا
        case '\uFB50', '\uFB51': // Yeh with hamza above (isolated/final)
                return '\u0626'
        case '\uFB52', '\uFB53', '\uFB54', '\uFB55': // Beh (all positional forms)
                return '\u0628'
        case '\uFB56', '\uFB57', '\uFB58', '\uFB59': // Peh
                return '\u067E'
        case '\uFB5A', '\uFB5B', '\uFB5C', '\uFB5D': // Tcheh
                return '\u0686'
        case '\uFB5E', '\uFB5F', '\uFB60', '\uFB61': // Jeh
                return '\u0698'
        case '\uFB62', '\uFB63', '\uFB64', '\uFB65': // Keheh
                return '\u06A9'
        case '\uFB66', '\uFB67', '\uFB68', '\uFB69': // Gaf
                return '\u06AF'
        case '\uFB6A', '\uFB6B', '\uFB6C', '\uFB6D': // Noon ghunna
                return '\u06BA'
        case '\uFB6E', '\uFB6F', '\uFB70', '\uFB71': // Heh doelachashm
                return '\u06BE'
        case '\uFB72', '\uFB73', '\uFB74', '\uFB75': // Farsi yeh
                return '\u06CC'
        case '\uFB76', '\uFB77', '\uFB78', '\uFB79': // Heh yeh
                return '\u06D5'
        default:
                return r
        }
}

// mapArabicPresentationB maps Arabic Presentation Forms-B (FE70-FEFF) to base forms.
func mapArabicPresentationB(r rune) rune {
        switch {
        case r >= '\uFE70' && r <= '\uFE7F':
                // Diacritic combinations - return space (will be collapsed later)
                return ' '
        case r == '\uFE80': // Hamza isolated
                return '\u0621'
        case r == '\uFE81' || r == '\uFE82': // Alef with madd above
                return '\u0622'
        case r == '\uFE83' || r == '\uFE84': // Alef with hamza above
                return '\u0623'
        case r == '\uFE85' || r == '\uFE86': // Waw with hamza above
                return '\u0624'
        case r == '\uFE87' || r == '\uFE88': // Alef with hamza below
                return '\u0625'
        case r == '\uFE89' || r == '\uFE8A' || r == '\uFE8B' || r == '\uFE8C': // Yeh with hamza above
                return '\u0626'
        case r == '\uFE8D' || r == '\uFE8E': // Alef
                return '\u0627'
        case r == '\uFE8F' || r == '\uFE90' || r == '\uFE91' || r == '\uFE92': // Beh
                return '\u0628'
        case r == '\uFE93' || r == '\uFE94': // Teh marbouta
                return '\u0629'
        case r == '\uFE95' || r == '\uFE96' || r == '\uFE97' || r == '\uFE98': // Theh (ت)
                return '\u062A'
        case r == '\uFE99' || r == '\uFE9A' || r == '\uFE9B' || r == '\uFE9C': // Theh (ث)
                return '\u062B'
        case r == '\uFE9D' || r == '\uFE9E' || r == '\uFE9F' || r == '\uFEA0': // Jeem
                return '\u062C'
        case r == '\uFEA1' || r == '\uFEA2' || r == '\uFEA3' || r == '\uFEA4': // Hah (ح)
                return '\u062D'
        case r == '\uFEA5' || r == '\uFEA6' || r == '\uFEA7' || r == '\uFEA8': // Khah (خ)
                return '\u062E'
        case r == '\uFEA9' || r == '\uFEAA': // Dal
                return '\u062F'
        case r == '\uFEAB' || r == '\uFEAC': // Thal (ذ)
                return '\u0630'
        case r == '\uFEAD' || r == '\uFEAE': // Reh (ر)
                return '\u0631'
        case r == '\uFEAF' || r == '\uFEB0': // Zain (ز)
                return '\u0632'
        case r == '\uFEB1' || r == '\uFEB2' || r == '\uFEB3' || r == '\uFEB4': // Seen
                return '\u0633'
        case r == '\uFEB5' || r == '\uFEB6' || r == '\uFEB7' || r == '\uFEB8': // Sheen
                return '\u0634'
        case r == '\uFEB9' || r == '\uFEBA' || r == '\uFEBB' || r == '\uFEBC': // Sad
                return '\u0635'
        case r == '\uFEBD' || r == '\uFEBE' || r == '\uFEBF' || r == '\uFEC0': // Dad
                return '\u0636'
        case r == '\uFEC1' || r == '\uFEC2' || r == '\uFEC3' || r == '\uFEC4': // Tah
                return '\u0637'
        case r == '\uFEC5' || r == '\uFEC6' || r == '\uFEC7' || r == '\uFEC8': // Zah
                return '\u0638'
        case r == '\uFEC9' || r == '\uFECA' || r == '\uFECB' || r == '\uFECC': // Ain
                return '\u0639'
        case r == '\uFECD' || r == '\uFECE' || r == '\uFECF' || r == '\uFED0': // Ghain
                return '\u063A'
        case r == '\uFED1' || r == '\uFED2' || r == '\uFED3' || r == '\uFED4': // Feh
                return '\u0641'
        case r == '\uFED5' || r == '\uFED6' || r == '\uFED7' || r == '\uFED8': // Qaf
                return '\u0642'
        case r == '\uFED9' || r == '\uFEDA' || r == '\uFEDB' || r == '\uFEDC': // Kaf
                return '\u0643'
        case r == '\uFEDD' || r == '\uFEDE' || r == '\uFEDF' || r == '\uFEE0': // Lam
                return '\u0644'
        case r == '\uFEE1' || r == '\uFEE2' || r == '\uFEE3' || r == '\uFEE4': // Meem
                return '\u0645'
        case r == '\uFEE5' || r == '\uFEE6' || r == '\uFEE7' || r == '\uFEE8': // Noon
                return '\u0646'
        case r == '\uFEE9' || r == '\uFEEA' || r == '\uFEEB' || r == '\uFEEC': // Heh
                return '\u0647'
        case r == '\uFEED' || r == '\uFEEE': // Waw
                return '\u0648'
        case r == '\uFEEF' || r == '\uFEF0': // Alef maksura -> ي
                return '\u064A'
        case r == '\uFEF1' || r == '\uFEF2' || r == '\uFEF3' || r == '\uFEF4': // Yeh
                return '\u064A'
        case r == '\uFEF5' || r == '\uFEF6' || r == '\uFEF7' || r == '\uFEF8' || // Lam-Alef ligatures
                r == '\uFEF9' || r == '\uFEFA' || r == '\uFEFB' || r == '\uFEFC':
                return '\u0644' // Lam
        default:
                return r
        }
}

// collapseSpaces reduces multiple consecutive spaces to a single space.
func collapseSpaces(s string) string {
        var b strings.Builder
        b.Grow(len(s))
        prevSpace := false

        for _, r := range s {
                if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
                        if !prevSpace {
                                b.WriteRune(' ')
                                prevSpace = true
                        }
                } else {
                        b.WriteRune(r)
                        prevSpace = false
                }
        }

        return b.String()
}

// ArabicUnicodeBlock returns true if the string contains any Arabic script characters.
func ArabicUnicodeBlock(s string) bool {
        for _, r := range s {
                if isArabicChar(r) {
                        return true
                }
        }
        return false
}

// isArabicChar checks if a rune belongs to an Arabic Unicode block.
func isArabicChar(r rune) bool {
        return (r >= arabicBlockStart && r <= arabicBlockEnd) ||
                (r >= arabicSupplement && r <= arabicSupplementEnd) ||
                (r >= arabicExtendedAStart && r <= arabicExtendedAEnd) ||
                (r >= arabicPresentationA && r <= arabicPresentationAEnd) ||
                (r >= arabicPresentationB && r <= arabicPresentationBEnd)
}
