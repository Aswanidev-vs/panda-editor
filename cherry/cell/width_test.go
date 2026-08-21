package cell

import "testing"

func TestRuneWidth(t *testing.T) {
	cases := []struct {
		r    rune
		want int
	}{
		{'a', 1},
		{'Z', 1},
		{'~', 1},
		{0x00E9, 1},          // é precomposed, ambiguous-free Latin-1
		{'漢', 2},             // CJK unified
		{'한', 2},             // hangul syllable
		{'｡', 1},             // halfwidth katakana period (FF61) — not in FF00-FF60
		{'Ａ', 2},             // fullwidth A (FF21)
		{'\u2665', 1},        // ♥ ambiguous -> 1
		{'😀', 2},             // U+1F600 emoticon
		{'🚀', 2},             // U+1F680
		{'\u1100', 2},        // hangul jamo cho
		{'あ', 2},             // hiragana
		{'\uFF01', 2},        // fullwidth !
		{'\u0301', 0},        // combining acute
		{'\u200B', 0},        // zero width space
		{'\u200D', 0},        // ZWJ
		{'\uFE0F', 0},        // VS16
		{'\uFEFF', 0},        // BOM
		{'\u20E3', 0},        // combining enclosing keycap
		{'\u0007', 0},        // bell (C0)
		{'\u007F', 0},        // DEL
		{'\u0085', 0},        // NEL (C1)
		{'\U0002A700', 2},    // cjk ext B
	}
	for _, c := range cases {
		if got := RuneWidth(c.r); got != c.want {
			t.Errorf("RuneWidth(%q) = %d, want %d", c.r, got, c.want)
		}
	}
}

func TestStringWidth(t *testing.T) {
	cases := []struct {
		s    string
		want int
	}{
		{"", 0},
		{"hello", 5},
		{"漢字", 4},
		// base + combining mark: the base occupies one column; the mark is
		// dropped by cell-based layout, so total reported width is 1.
		{"e\u0301", 1},
		{"café", 4},
	}
	for _, c := range cases {
		if got := StringWidth(c.s); got != c.want {
			t.Errorf("StringWidth(%q) = %d, want %d", c.s, got, c.want)
		}
	}
}

// TestVS16Limitation documents the v1 stance on variation selector sequences:
// VS16 itself is zero-width and the sequence counts at its base rune's width,
// even though some terminals render e.g. U+2764 U+FE0F double-wide.
func TestVS16Limitation(t *testing.T) {
	const heart = "\u2764\uFE0F"
	if got := StringWidth(heart); got != 1 {
		t.Errorf("StringWidth(%q) = %d, want 1 (documented VS16 limitation)", heart, got)
	}
}

func TestRuneWidthTableSorted(t *testing.T) {
	for name, table := range map[string][]widthRange{
		"zeroWidth": zeroWidth,
		"wide":      wideRanges,
	} {
		for i := 1; i < len(table); i++ {
			if table[i].lo <= table[i-1].hi {
				t.Errorf("%s: range %v overlaps or precedes %v", name, table[i-1], table[i])
			}
		}
	}
}
