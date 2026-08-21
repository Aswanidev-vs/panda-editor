package textbuf

import (
	"slices"
	"strings"
	"testing"
)

func TestNewEmpty(t *testing.T) {
	b := New()
	if b.LineCount() != 1 || b.Line(0) != "" || b.Len() != 0 || b.Text() != "" {
		t.Fatalf("fresh buffer: lines=%d line=%q len=%d text=%q", b.LineCount(), b.Line(0), b.Len(), b.Text())
	}
	if b.LineLen(0) != 0 {
		t.Errorf("LineLen(0) = %d, want 0", b.LineLen(0))
	}
	if b.Modified() {
		t.Error("New buffer should not be modified")
	}
}

// The old internal/buffer normalized CRLF on load; v2 keeps \r verbatim.
func TestFromStringKeepsCRVerbatim(t *testing.T) {
	b := FromString("alpha\r\nbeta\r\ngamma")
	if got := b.Text(); got != "alpha\r\nbeta\r\ngamma" {
		t.Fatalf("Text() = %q, want verbatim input", got)
	}
	if b.LineCount() != 3 {
		t.Fatalf("LineCount = %d, want 3", b.LineCount())
	}
	if b.Line(0) != "alpha\r" || b.Line(1) != "beta\r" || b.Line(2) != "gamma" {
		t.Errorf("lines = %q, %q, %q; CR must survive as content", b.Line(0), b.Line(1), b.Line(2))
	}
	if got := b.DominantEOL(); got != EOLCRLF {
		t.Errorf("DominantEOL = %v, want EOLCRLF", got)
	}
}

func TestInsertAndDeleteClamping(t *testing.T) {
	b := New()
	b.Insert(-10, "ab")
	b.Insert(1<<30, "cd")
	if got := b.Text(); got != "abcd" {
		t.Fatalf("Text after clamped inserts = %q, want %q", got, "abcd")
	}
	b.DeleteRange(50, 100) // fully out of range: no-op
	if got := b.Text(); got != "abcd" {
		t.Fatalf("out-of-range delete changed text: %q", got)
	}
	b.DeleteRange(3, 1) // reversed ends are swapped
	if got := b.Text(); got != "ad" {
		t.Fatalf("after reversed delete = %q, want %q", got, "ad")
	}
}

func TestDeletePosForward(t *testing.T) {
	b := FromString("abc\ndef")
	b.DeletePos(Pos{Line: 0, Col: 2}, 2) // eats 'c' and the newline
	if got := b.Text(); got != "abdef" {
		t.Fatalf("DeletePos across newline = %q, want %q", got, "abdef")
	}
	b.DeletePos(Pos{Line: 0, Col: 4}, 100) // oversized n clamps at document end, eating 'f'
	if got := b.Text(); got != "abde" {
		t.Fatalf("oversized DeletePos = %q, want %q", got, "abde")
	}
	b.DeletePos(Pos{Line: 0, Col: 0}, 0)
	b.DeletePos(Pos{Line: 0, Col: 0}, -3)
	b.DeletePos(Pos{Line: 0, Col: b.LineLen(0)}, 1) // anchored at doc end: no-op
	if got := b.Text(); got != "abde" {
		t.Fatalf("zero/negative/end DeletePos changed text: %q", got)
	}
}

// White-box: incremental index maintenance must always agree with a fresh
// scan of the content, for both the incremental and rebuild code paths.
func TestLineIndexIncrementalMatchesRescan(t *testing.T) {
	b := FromString("one\ntwo\nthree")
	ops := []func(){
		func() { b.Insert(4, "X") },
		func() { b.Insert(7, "\n") },         // single newline insert
		func() { b.Insert(0, "\n") },         // newline at very front
		func() { b.Insert(b.Len(), "y\nz") }, // trailing partial line
		func() { b.Insert(5, "\n\n") },       // multi-newline: rebuild path
		func() { b.DeleteRange(0, 1) },
		func() { b.DeleteRange(3, 9) }, // multi-newline delete: rebuild path
		func() { b.DeleteRange(b.Len()-1, b.Len()) },
		func() { b.Insert(2, "q\nw") },
		func() { b.DeleteRange(0, b.Len()) },
		func() { b.Insert(0, "a\nb\nc") },
	}
	for i, op := range ops {
		op()
		want := rescanLineStarts(b)
		if !slices.Equal(b.lineStarts, want) {
			t.Fatalf("after op %d (text=%q): lineStarts = %v, want %v", i, b.Text(), b.lineStarts, want)
		}
		if b.LineCount() != len(want) {
			t.Fatalf("op %d: LineCount = %d, want %d", i, b.LineCount(), len(want))
		}
		for ln := 0; ln < b.LineCount(); ln++ {
			if b.LineLen(ln) < 0 {
				t.Fatalf("op %d: negative LineLen(%d)", i, ln)
			}
		}
	}
}

// rescanLineStarts recomputes line starts from Text(), independently of the
// incremental machinery under test.
func rescanLineStarts(tb *Buffer) []int {
	starts := []int{0}
	s := tb.Text()
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

func TestPosConversions(t *testing.T) {
	// bytes: a|日(1..3)|b|c|\n(6)|で(7..9)|\n(10)|x|f ⇒ Len 13, starts [0,7,11]
	b := FromString("a日bc\nで\nxf")
	toOff := map[Pos]int{
		{Line: 0, Col: 0}:   0,
		{Line: 0, Col: 1}:   1,
		{Line: 0, Col: 2}:   4,
		{Line: 0, Col: 3}:   5,
		{Line: 0, Col: 4}:   6, // end of line 0 (before '\n')
		{Line: 0, Col: 9}:   6,
		{Line: 1, Col: 0}:   7,
		{Line: 1, Col: 1}:   10,
		{Line: 1, Col: 7}:   10,
		{Line: 2, Col: 0}:   11,
		{Line: 2, Col: 2}:   13,
		{Line: 2, Col: 5}:   13,
		{Line: 9, Col: 0}:   11, // line clamped
		{Line: -4, Col: 0}:  0,
		{Line: -4, Col: -2}: 0,
	}
	for p, want := range toOff {
		if got := b.PosToOffset(p); got != want {
			t.Errorf("PosToOffset(%+v) = %d, want %d", p, got, want)
		}
	}
	toPos := map[int]Pos{
		0:  {Line: 0, Col: 0},
		1:  {Line: 0, Col: 1},
		2:  {Line: 0, Col: 1}, // mid-rune snaps back
		3:  {Line: 0, Col: 1},
		4:  {Line: 0, Col: 2},
		5:  {Line: 0, Col: 3},
		6:  {Line: 0, Col: 4},
		7:  {Line: 1, Col: 0},
		8:  {Line: 1, Col: 0}, // mid-rune snaps back
		9:  {Line: 1, Col: 0},
		10: {Line: 1, Col: 1},
		11: {Line: 2, Col: 0},
		12: {Line: 2, Col: 1},
		13: {Line: 2, Col: 2},
		99: {Line: 2, Col: 2}, // clamped
	}
	for off, want := range toPos {
		if got := b.OffsetToPos(off); got != want {
			t.Errorf("OffsetToPos(%d) = %+v, want %+v", off, got, want)
		}
	}
	for off := 0; off <= b.Len(); off++ {
		p := b.OffsetToPos(off)
		if back := b.PosToOffset(p); back != b.snapBack(off) {
			t.Errorf("roundtrip %d: PosToOffset(%+v) = %d, want %d", off, p, back, b.snapBack(off))
		}
	}
}

func TestInsertRuneIntoChoppedRune(t *testing.T) {
	b := FromString("日本語") // 日 = E6 97 A5
	b.DeleteRange(0, 2)    // leaves stray continuation byte 0xA5 in front
	if got := b.Text(); got != "\xa5本語" {
		t.Fatalf("setup: Text = %q", got)
	}
	// Col 1 lands on 本's lead byte — already a boundary, so X goes after
	// the stray byte without snapping.
	b.InsertRune(Pos{Line: 0, Col: 1}, 'X')
	if got := b.Text(); got != "\xa5X本語" {
		t.Fatalf("after InsertRune into chopped rune = %q, want %q", got, "\xa5X本語")
	}
	// Chop 本's lead byte: two stray continuation bytes now follow X.
	// Targeting their column must snapBack left onto the boundary after X.
	b.DeleteRange(2, 3)
	if got := b.Text(); got != "\xa5X\x9c\xac語" {
		t.Fatalf("setup2: Text = %q", got)
	}
	b.InsertRune(Pos{Line: 0, Col: 2}, 'Y')
	if got := b.Text(); got != "\xa5YX\x9c\xac語" {
		t.Fatalf("snapBack insert = %q, want %q", got, "\xa5YX\x9c\xac語")
	}
}

func TestGapGrowthAndSeek(t *testing.T) {
	b := New()
	for i := 0; i < 2000; i++ {
		b.Insert(0, "x") // prepending forces gap seeks and capacity growth
	}
	for i := 0; i < 2000; i++ {
		b.Insert(b.Len()/2, "y")
	}
	if b.Len() != 4000 || b.LineCount() != 1 {
		t.Fatalf("Len=%d LineCount=%d, want 4000/1", b.Len(), b.LineCount())
	}
	if got := strings.Count(b.Text(), "x"); got != 2000 {
		t.Fatalf("x count = %d, want 2000", got)
	}
	if got := strings.Count(b.Text(), "y"); got != 2000 {
		t.Fatalf("y count = %d, want 2000", got)
	}
	line := b.Line(0)
	if len(line) != 4000 {
		t.Fatalf("Line(0) length = %d, want 4000", len(line))
	}
	b.DeleteRange(0, 1500)
	b.DeleteRange(500, 2500)
	if b.Len() != 500 {
		t.Fatalf("Len after chunk deletes = %d, want 500", b.Len())
	}
}

func TestLineOutOfBounds(t *testing.T) {
	b := FromString("one")
	if b.Line(-1) != "" || b.Line(1) != "" || b.Line(99) != "" {
		t.Error("out-of-range Line should return empty string")
	}
	if b.LineLen(-1) != 0 || b.LineLen(1) != 0 {
		t.Error("out-of-range LineLen should return 0")
	}
	if b.LineCount() != 1 {
		t.Errorf("LineCount = %d, want 1", b.LineCount())
	}
}

func TestModifiedLifecycle(t *testing.T) {
	b := New()
	b.Insert(0, "x")
	if !b.Modified() {
		t.Fatal("edit should mark buffer modified")
	}
	b.SetModified(false)
	if b.Modified() {
		t.Fatal("SetModified(false) should clear dirty state")
	}
	b.Insert(b.Len(), "y")
	if !b.Modified() {
		t.Fatal("edit after sync should mark buffer modified again")
	}
	fresh := New()
	fresh.SetModified(true)
	if !fresh.Modified() {
		t.Fatal("SetModified(true) should force dirty even with no edits")
	}
}

func TestEmptyOpsAreNoops(t *testing.T) {
	b := FromString("keep")
	before := b.edits
	b.Insert(2, "")
	b.DeleteRange(1, 1)
	b.DeletePos(Pos{Line: 0, Col: 1}, 0)
	if b.edits != before {
		t.Fatalf("empty ops bumped edit counter: %d -> %d", before, b.edits)
	}
	if got := b.Text(); got != "keep" {
		t.Fatalf("text changed by empty ops: %q", got)
	}
}
