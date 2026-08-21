package cell

// Rune display width, from scratch. Tables are compact, sorted, non-overlapping
// [lo,hi] ranges; lookup is a binary search. Ambiguous-width runes resolve to 1.
//
// Known v1 limitations (deliberate, documented):
//   - VS16 (\uFE0F) emoji text-presentation sequences count at the base rune's
//     width; the selector itself is zero-width, so "♥\uFE0F" is reported as 1
//     even where the terminal draws it double-wide.
//   - ZWJ emoji sequences (family flags etc.) sum member widths instead of
//     collapsing to one glyph.

type widthRange struct{ lo, hi rune }

// zeroWidth covers combining marks (Mn/Me), format characters (Cf) and other
// default-ignorable code points that occupy no terminal column.
var zeroWidth = []widthRange{
	{0x0300, 0x036F}, // combining diacritical marks
	{0x0483, 0x0489}, // combining cyrillic
	{0x0591, 0x05BD}, // hebrew points
	{0x05BF, 0x05BF},
	{0x05C1, 0x05C2},
	{0x05C4, 0x05C5},
	{0x05C7, 0x05C7},
	{0x0610, 0x061A}, // arabic combining
	{0x061C, 0x061C}, // arabic letter mark
	{0x064B, 0x065F}, // arabic diacritics
	{0x0670, 0x0670},
	{0x06D6, 0x06DC},
	{0x06DF, 0x06E4},
	{0x06E7, 0x06E8},
	{0x06EA, 0x06ED},
	{0x070F, 0x070F}, // syriac abbreviation mark
	{0x0711, 0x0711},
	{0x0730, 0x074A},
	{0x07A6, 0x07B0}, // thaana combining
	{0x07EB, 0x07F3}, // nko combining
	{0x0816, 0x0819}, // samaritan combining
	{0x081B, 0x0823},
	{0x0825, 0x0827},
	{0x0829, 0x082D},
	{0x0859, 0x085B}, // mandaic combining
	{0x08E2, 0x08E2}, // arabic number sign above
	{0x093C, 0x093C}, // devanagari nukta
	{0x0951, 0x0957}, // vedic tone marks
	{0x0E31, 0x0E31}, // thai
	{0x0E34, 0x0E3A},
	{0x0E47, 0x0E4E},
	{0x0EB1, 0x0EB1}, // lao
	{0x0EB4, 0x0EBC},
	{0x0EC8, 0x0ECD},
	{0x180B, 0x180F}, // mongolian free variation selectors + vowel separator
	{0x1AB0, 0x1AFF}, // combining diacritical marks extended
	{0x1DC0, 0x1DFF}, // combining diacritical marks supplement
	{0x200B, 0x200F}, // ZWSP, ZWNJ, ZWJ, LRM, RLM
	{0x202A, 0x202E}, // bidi embedding controls
	{0x2060, 0x2064}, // word joiner, invisible operators
	{0x2066, 0x206F}, // bidi isolates and deprecated formats
	{0x20D0, 0x20F0}, // combining marks for symbols
	{0x2CEF, 0x2CF1}, // coptic combining
	{0x2DE0, 0x2DFF}, // cyrillic extended-A combining
	{0xA66F, 0xA672}, // coptic combining
	{0xA674, 0xA67D},
	{0xFE00, 0xFE0F}, // variation selectors (VS1-VS16)
	{0xFE20, 0xFE2F}, // combining half marks
	{0xFEFF, 0xFEFF}, // zero width no-break space / BOM
	{0xFFF9, 0xFFFB}, // interlinear annotation anchors
	{0x101FD, 0x101FD},
	{0x102E0, 0x102E0},
	{0x10376, 0x1037A},
	{0x1D173, 0x1D17A}, // musical format characters
	{0xE0001, 0xE0001}, // language tag
	{0xE0020, 0xE007F}, // tag characters
	{0xE0100, 0xE01EF}, // variation selectors supplement
}

// wide covers East Asian Wide (W) and Fullwidth (F) code points plus the
// emoji-presentation core. Emoji ranges here intentionally cover whole blocks;
// ambiguous width resolves to 1.
var wideRanges = []widthRange{
	{0x1100, 0x115F}, // hangul jamo
	{0x231A, 0x231B}, // watch, hourglass
	{0x2329, 0x232A}, // angle brackets
	{0x23E9, 0x23EC},
	{0x23F0, 0x23F0},
	{0x23F3, 0x23F3},
	{0x25FD, 0x25FE},
	{0x2614, 0x2615},
	{0x2648, 0x2653},
	{0x267F, 0x267F},
	{0x2693, 0x2693},
	{0x26A1, 0x26A1},
	{0x26AA, 0x26AB},
	{0x26BD, 0x26BE},
	{0x26C4, 0x26C5},
	{0x26CE, 0x26CE},
	{0x26D4, 0x26D4},
	{0x26EA, 0x26EA},
	{0x26F2, 0x26F3},
	{0x26F5, 0x26F5},
	{0x26FA, 0x26FA},
	{0x26FD, 0x26FD},
	{0x2705, 0x2705},
	{0x270A, 0x270B},
	{0x2728, 0x2728},
	{0x274C, 0x274C},
	{0x274E, 0x274E},
	{0x2753, 0x2755},
	{0x2757, 0x2757},
	{0x2795, 0x2797},
	{0x27B0, 0x27B0},
	{0x27BF, 0x27BF},
	{0x2B1B, 0x2B1C},
	{0x2B50, 0x2B50},
	{0x2B55, 0x2B55},
	{0x2E80, 0x303E}, // cjk radicals through cjk symbols
	{0x3041, 0x33FF}, // hiragana through cjk compatibility
	{0x3400, 0x4DBF}, // cjk extension A
	{0x4E00, 0x9FFF}, // cjk unified ideographs
	{0xA000, 0xA4CF}, // yi syllables
	{0xA960, 0xA97F}, // hangul jamo extended-A
	{0xAC00, 0xD7A3}, // hangul syllables
	{0xF900, 0xFAFF}, // cjk compatibility ideographs
	{0xFE10, 0xFE19}, // vertical forms
	{0xFE30, 0xFE4F}, // cjk compatibility forms
	{0xFF00, 0xFF60}, // fullwidth forms
	{0xFFE0, 0xFFE6}, // fullwidth signs
	{0x1F004, 0x1F004}, // mahjong red dragon
	{0x1F0CF, 0x1F0CF}, // playing card black joker
	{0x1F18E, 0x1F18E}, // ab button
	{0x1F191, 0x1F19A}, // squared emoji
	{0x1F300, 0x1F5FF}, // misc symbols and pictographs
	{0x1F600, 0x1F64F}, // emoticons
	{0x1F680, 0x1F6FF}, // transport and map
	{0x1F900, 0x1F9FF}, // supplemental symbols and pictographs
	{0x20000, 0x2FFFD}, // cjk extensions B-F
	{0x30000, 0x3FFFD}, // cjk extensions G+
}

func inRanges(r rune, table []widthRange) bool {
	lo, hi := 0, len(table)-1
	for lo <= hi {
		mid := int(uint(lo+hi) >> 1)
		switch {
		case r < table[mid].lo:
			hi = mid - 1
		case r > table[mid].hi:
			lo = mid + 1
		default:
			return true
		}
	}
	return false
}

// RuneWidth reports the number of terminal columns r occupies:
// 0 for combining marks and format/control characters, 2 for East Asian
// Wide/Fullwidth and emoji-presentation runes, 1 otherwise. Ambiguous-width
// runes are 1. Combining marks are expected to be dropped by callers that lay
// out text cell by cell (see Screen.Print).
func RuneWidth(r rune) int {
	if r < 0x20 || (r >= 0x7F && r < 0xA0) {
		return 0 // C0/C1 controls have no defined presentation
	}
	if inRanges(r, zeroWidth) {
		return 0
	}
	if inRanges(r, wideRanges) {
		return 2
	}
	return 1
}

// StringWidth sums RuneWidth over s's runes.
func StringWidth(s string) int {
	n := 0
	for _, r := range s {
		n += RuneWidth(r)
	}
	return n
}
