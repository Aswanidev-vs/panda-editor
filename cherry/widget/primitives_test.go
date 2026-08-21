package widget

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Aswanidev-vs/cherry/cell"
	"github.com/Aswanidev-vs/cherry/geom"
)

func TestWrapText(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		width int
		want  []string
	}{
		{"zero width passthrough", "hello world", 0, []string{"hello world"}},
		{"negative width passthrough", "abc", -3, []string{"abc"}},
		{"no wrap needed", "hello world", 20, []string{"hello world"}},
		{"exact fit boundary", "ab cd ef", 8, []string{"ab cd ef"}},
		{"greedy fill", "aa bb cc dd", 5, []string{"aa bb", "cc dd"}},
		{"word longer than line", "aaaa bb", 2, []string{"aa", "aa", "bb"}},
		{"oversized word keeps tail open", "aaa b cc", 3, []string{"aaa", "b", "cc"}},
		{"explicit newline preserved", "a\nb", 10, []string{"a", "b"}},
		{"empty input single blank line", "", 5, []string{""}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := WrapText(tc.in, tc.width)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("WrapText(%q,%d) = %q, want %q", tc.in, tc.width, got, tc.want)
			}
		})
	}
}

func TestWrapTextWideRunes(t *testing.T) {
	// 漢字 is width 4; width-6 line fits "漢字 x" (4+1+1).
	got := WrapText("漢字 x y", 6)
	if len(got) != 2 || got[0] != "漢字 x" || got[1] != "y" {
		t.Fatalf("wide-rune wrap = %q", got)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s, tail string
		width   int
		want    string
	}{
		{"short", "…", 10, "short"},
		{"exactly fits", "abcdef", 6, "abcdef"},
		{"abcdef", "…", 4, "abc…"},
		{"abcdef", "......", 2, "ab"},
		{"abcdef", "", 0, ""},
	}
	for _, tc := range tests {
		if got := Truncate(tc.s, tc.width, tc.tail); got != tc.want {
			t.Errorf("Truncate(%q,%d,%q) = %q, want %q", tc.s, tc.width, tc.tail, got, tc.want)
		}
	}
	if got := Truncate("漢字漢字", 5, "…"); strWidth(got) > 5 {
		t.Errorf("multibyte truncate overflowed budget: %q (%d cols)", got, strWidth(got))
	}
}

func TestStatusSlots(t *testing.T) {
	natural := func(txt string) Segment { return Segment{Text: txt} }
	flex := func(n int) Segment { return Segment{Flex: n} }

	// Natural-only slots equal their text widths.
	got := statusSlots([]Segment{natural("ab"), natural("cdef")}, 80)
	if !reflect.DeepEqual(got, []int{2, 4}) {
		t.Fatalf("natural slots = %v", got)
	}

	// Flex splits leftover; remainders go to later flex segments.
	got = statusSlots([]Segment{natural("ab"), flex(1), flex(2)}, 20)
	if sum(got) != 20 {
		t.Fatalf("flex slots %v do not sum to width", got)
	}
	if got[0] != 2 || got[1]+got[2] != 18 {
		t.Fatalf("flex distribution wrong: %v", got)
	}

	// Overflowing natural widths are clamped at draw time; slots just report.
	got = statusSlots([]Segment{natural("aaaaaaaa"), natural("bbbbbbbb")}, 4)
	if got[0] != 8 || got[1] != 8 {
		t.Fatalf("overflow slots = %v", got)
	}

	// Zero flex weight behaves as natural.
	got = statusSlots([]Segment{flex(0), natural("x")}, 10)
	if got[1] != 1 {
		t.Fatalf("zero-flex slot = %v", got)
	}
}

func TestProgressCells(t *testing.T) {
	tests := []struct {
		barW  int
		ratio float64
		want  int
	}{
		{10, 0, 0}, {10, 1, 10}, {10, 0.5, 5}, {10, 0.55, 5}, {3, 0.99, 2},
	}
	for _, tc := range tests {
		if got := filledCells(tc.barW, tc.ratio); got != tc.want {
			t.Errorf("filledCells(%d,%v) = %d, want %d", tc.barW, tc.ratio, got, tc.want)
		}
	}

	p := &Progress{Value: 1.7}
	if p.ratio() != 1 {
		t.Error("ratio must clamp above 1")
	}
	p.Value = -3
	if p.ratio() != 0 {
		t.Error("ratio must clamp below 0")
	}
}

func TestBoxMeasure(t *testing.T) {
	child := &Spacer{MinW: 10, MinH: 4}
	b := &Box{Mode: BorderSingle, PadL: 2, PadR: 1, PadT: 1, PadB: 0, Child: child}
	got := b.Measure(geom.Size{})
	if got.W != 1+2+10+1+1 || got.H != 1+1+4+0+1 {
		t.Fatalf("Box measure with chrome = %v, want {15,7}", got)
	}

	bounded := b.Measure(geom.Size{W: 9, H: 9})
	if bounded.W != 9 || bounded.H != 7 {
		t.Fatalf("Box measure clamped = %v, want {9 7}", bounded)
	}

	titleOnly := &Box{Mode: BorderRounded, Title: "hey"}
	m := titleOnly.Measure(geom.Size{})
	if m.W != strWidth(" hey ")+2 || m.H != 2 {
		t.Fatalf("title-only box = %v", m)
	}

	borderless := &Box{Child: child}
	if got := borderless.Measure(geom.Size{}); got.W != 10 || got.H != 4 {
		t.Fatalf("BorderNone adds no chrome: %v", got)
	}
}

func TestSpacerMeasure(t *testing.T) {
	s := &Spacer{MinW: 3, MinH: 2, FillX: true}
	if got := s.Measure(geom.Size{W: 40, H: 10}); got.W != 40 || got.H != 2 {
		t.Fatalf("FillX spacer = %v", got)
	}
	if got := s.Measure(geom.Size{}); got.W != 3 || got.H != 2 {
		t.Fatalf("unconstrained spacer = %v", got)
	}
	fixed := &Spacer{}
	if got := fixed.Measure(geom.Size{W: 5, H: 5}); got.W != 0 || got.H != 0 {
		t.Fatalf("empty spacer should be zero-sized: %v", got)
	}
}

func TestSpinnerTickWraps(t *testing.T) {
	s := NewSpinner(FramesLine, cellPlain())
	for i := 0; i < len(FramesLine); i++ {
		s.Tick()
	}
	if s.Frame != 0 {
		t.Fatalf("Tick did not wrap: %d", s.Frame)
	}
	empty := &Spinner{}
	empty.Tick() // must not panic or move
	if empty.Frame != 0 {
		t.Error("empty spinner tick moved frame")
	}
	if f := empty.frame(); f != "" {
		t.Errorf("empty spinner frame = %q", f)
	}
}

func TestTextMeasureAndAlign(t *testing.T) {
	tx := &Text{Content: "one\ntwo\nthree"}
	if got := tx.Measure(geom.Size{}); got.W != 5 || got.H != 3 {
		t.Fatalf("plain Text measure = %v", got)
	}
	wrapped := &Text{Content: strings.Repeat("word ", 10), Wrap: true}
	if got := wrapped.Measure(geom.Size{W: 10, H: 0}); got.W > 10 {
		t.Fatalf("wrapped measure exceeded max width: %v", got)
	}
}

// cellPlain keeps the helper surface tiny for tests that need a Style.
func cellPlain() cell.Style { return cell.Style{} }

// sum totals a slot slice.
func sum(xs []int) int {
	total := 0
	for _, v := range xs {
		total += v
	}
	return total
}
