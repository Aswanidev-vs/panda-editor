package document

import (
	"os"
	"testing"

	"github.com/Aswanidev-vs/panda-editor/editor/textbuf"
)

// newDoc builds a document holding text, untouched by Open.
func newDoc(text string) *Document {
	return &Document{buf: textbuf.FromString(text)}
}

// createFile writes content to path via os.Create and hands back the file.
func createFile(t *testing.T, path, content string) *os.File {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return f
}

func mustPos(t *testing.T, got, want Pos, what string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: cursor %v, want %v", what, got, want)
	}
}

func TestNewOpenPath(t *testing.T) {
	d := New()
	if d.LineCount() != 1 || d.Text() != "" || d.Modified() || d.Path() != "" {
		t.Fatalf("New: empty doc, want 1 empty line, got %q (%d lines, path %q)",
			d.Text(), d.LineCount(), d.Path())
	}
	d.SetPath("/tmp/x.txt")
	if d.Path() != "/tmp/x.txt" {
		t.Fatalf("SetPath: %q", d.Path())
	}

	// A missing file yields an empty document that remembers the path.
	missing := t.TempDir() + "/absent.txt"
	d2, err := Open(missing)
	if err != nil {
		t.Fatalf("Open(missing): %v", err)
	}
	if d2.Path() != missing || d2.Text() != "" || d2.Modified() {
		t.Fatalf("Open(missing): path %q text %q modified %v", d2.Path(), d2.Text(), d2.Modified())
	}

	// An existing file loads its content unmodified.
	p := t.TempDir() + "/real.txt"
	f := createFile(t, p, "alpha\nbeta")
	f.Close()
	d3, err := Open(p)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if d3.Text() != "alpha\nbeta" || d3.Path() != p || d3.Modified() {
		t.Fatalf("Open: %q", d3.Text())
	}
	if d3.Cursor() != (Pos{Line: 0, Col: 0}) {
		t.Fatalf("Open: cursor %v", d3.Cursor())
	}
}

func TestSetCursorClamps(t *testing.T) {
	d := newDoc("ab\n日本語")
	d.SetCursor(Pos{Line: 99, Col: 99})
	mustPos(t, d.Cursor(), Pos{Line: 1, Col: 3}, "line/col clamp")
	d.SetCursor(Pos{Line: 0, Col: 1})
	// A plain SetCursor clears any selection.
	d.SelectTo(Pos{Line: 1, Col: 2})
	d.SetCursor(Pos{Line: 0, Col: 0})
	if _, _, ok := d.Selection(); ok {
		t.Fatal("SetCursor must clear the selection")
	}
}

func TestMoveLeftRightAcrossLines(t *testing.T) {
	d := newDoc("ab\ncd\ne")
	d.SetCursor(Pos{Line: 1, Col: 0})
	d.Move(MoveLeft, false)
	mustPos(t, d.Cursor(), Pos{Line: 0, Col: 2}, "left crosses into line end")
	d.Move(MoveRight, false)
	mustPos(t, d.Cursor(), Pos{Line: 1, Col: 0}, "right crosses to next line")
	d.Move(MoveLeft, false) // back to {0,2}
	d.Move(MoveRight, false)
	mustPos(t, d.Cursor(), Pos{Line: 1, Col: 0}, "right from line end")

	d.SetCursor(Pos{Line: 2, Col: 1}) // eol of the last line
	d.Move(MoveRight, false)
	mustPos(t, d.Cursor(), Pos{Line: 2, Col: 1}, "right clamps at doc end")
	// Left first steps to column 0, then crosses into the previous line.
	d.Move(MoveLeft, false)
	mustPos(t, d.Cursor(), Pos{Line: 2, Col: 0}, "left steps to column 0 first")
	d.Move(MoveLeft, false)
	mustPos(t, d.Cursor(), Pos{Line: 1, Col: 2}, "left crosses to previous line end")

	d.SetCursor(Pos{Line: 0, Col: 0})
	d.Move(MoveLeft, false)
	mustPos(t, d.Cursor(), Pos{Line: 0, Col: 0}, "left clamps at doc start")

	// Rune step across a multi-byte boundary.
	d2 := newDoc("aéb")
	d2.SetCursor(Pos{Line: 0, Col: 2})
	d2.Move(MoveLeft, false)
	mustPos(t, d2.Cursor(), Pos{Line: 0, Col: 1}, "left over multi-byte rune")
	d2.Move(MoveRight, false)
	mustPos(t, d2.Cursor(), Pos{Line: 0, Col: 2}, "right over multi-byte rune")
}

func TestMoveUpDownColumnMemory(t *testing.T) {
	d := newDoc("abcdef\nab\nabcdefgh")
	d.SetCursor(Pos{Line: 0, Col: 5})
	d.Move(MoveDown, false) // line 1 has only 2 runes
	mustPos(t, d.Cursor(), Pos{Line: 1, Col: 2}, "down clamps into short line")
	d.Move(MoveDown, false)
	mustPos(t, d.Cursor(), Pos{Line: 2, Col: 5}, "down recovers the remembered column")
	d.Move(MoveUp, false)
	mustPos(t, d.Cursor(), Pos{Line: 1, Col: 2}, "up clamps again")
	d.Move(MoveUp, false)
	mustPos(t, d.Cursor(), Pos{Line: 0, Col: 5}, "up recovers the remembered column")

	// A horizontal move resets the memory.
	d.SetCursor(Pos{Line: 2, Col: 7})
	d.Move(MoveUp, false)   // line 1 too short -> {1,2}
	d.Move(MoveLeft, false) // {1,1}: memory becomes 1
	d.Move(MoveUp, false)
	mustPos(t, d.Cursor(), Pos{Line: 0, Col: 1}, "horizontal move resets column memory")

	// Document edges never panic and hold the memory.
	d.SetCursor(Pos{Line: 0, Col: 4})
	d.Move(MoveUp, false)
	mustPos(t, d.Cursor(), Pos{Line: 0, Col: 4}, "up at doc start")
	d.Move(MoveDown, false)
	d.Move(MoveDown, false)
	d.Move(MoveDown, false)
	mustPos(t, d.Cursor(), Pos{Line: 2, Col: 4}, "down past doc end clamps to last line")

	// MoveLines jumps several rows at once.
	d.SetCursor(Pos{Line: 0, Col: 5})
	d.MoveLines(2, false)
	mustPos(t, d.Cursor(), Pos{Line: 2, Col: 5}, "MoveLines down")
	d.MoveLines(-2, false)
	mustPos(t, d.Cursor(), Pos{Line: 0, Col: 5}, "MoveLines up")
}

func TestMoveHomeEndDocBounds(t *testing.T) {
	d := newDoc("  abc  \nlast")
	d.SetCursor(Pos{Line: 0, Col: 4})
	d.Move(MoveHome, false)
	mustPos(t, d.Cursor(), Pos{Line: 0, Col: 0}, "home")
	d.Move(MoveEnd, false)
	mustPos(t, d.Cursor(), Pos{Line: 0, Col: 7}, "end")
	d.Move(MoveDocEnd, false)
	mustPos(t, d.Cursor(), Pos{Line: 1, Col: 4}, "doc end")
	d.Move(MoveDocStart, false)
	mustPos(t, d.Cursor(), Pos{Line: 0, Col: 0}, "doc start")
}

func TestMoveWord(t *testing.T) {
	d := newDoc("hello world foo")
	d.SetCursor(Pos{Line: 0, Col: 0})
	d.Move(MoveWordRight, false)
	mustPos(t, d.Cursor(), Pos{Line: 0, Col: 5}, "word right 1")
	d.Move(MoveWordRight, false)
	mustPos(t, d.Cursor(), Pos{Line: 0, Col: 11}, "word right 2")
	d.Move(MoveWordRight, false)
	mustPos(t, d.Cursor(), Pos{Line: 0, Col: 15}, "word right 3")
	d.Move(MoveWordRight, false)
	mustPos(t, d.Cursor(), Pos{Line: 0, Col: 15}, "word right clamps at doc end")
	d.Move(MoveWordLeft, false)
	mustPos(t, d.Cursor(), Pos{Line: 0, Col: 12}, "word left 1")
	d.Move(MoveWordLeft, false)
	mustPos(t, d.Cursor(), Pos{Line: 0, Col: 6}, "word left 2")
	d.Move(MoveWordLeft, false)
	mustPos(t, d.Cursor(), Pos{Line: 0, Col: 0}, "word left 3")
	d.Move(MoveWordLeft, false)
	mustPos(t, d.Cursor(), Pos{Line: 0, Col: 0}, "word left clamps at doc start")

	// Unicode letters are word runes; underscores are not.
	u := newDoc("漢字word _énd")
	u.SetCursor(Pos{Line: 0, Col: 0})
	u.Move(MoveWordRight, false)
	mustPos(t, u.Cursor(), Pos{Line: 0, Col: 6}, "unicode word right")
	u.Move(MoveWordLeft, false)
	mustPos(t, u.Cursor(), Pos{Line: 0, Col: 0}, "unicode word left")
}

func TestSelectToExtends(t *testing.T) {
	d := newDoc("abc\ndef")
	d.SetCursor(Pos{Line: 0, Col: 1})
	d.SelectTo(Pos{Line: 1, Col: 2})
	start, end, ok := d.Selection()
	if !ok || start != (Pos{Line: 0, Col: 1}) || end != (Pos{Line: 1, Col: 2}) {
		t.Fatalf("selection %v..%v ok=%v", start, end, ok)
	}
	if d.SelectionText() != "bc\nde" {
		t.Fatalf("SelectionText %q", d.SelectionText())
	}

	// Shrinking/extending keeps the anchor fixed.
	d.SelectTo(Pos{Line: 0, Col: 2})
	start, end, ok = d.Selection()
	if !ok || start != (Pos{Line: 0, Col: 1}) || end != (Pos{Line: 0, Col: 2}) {
		t.Fatalf("shrunk selection %v..%v", start, end)
	}
	if d.SelectionText() != "b" {
		t.Fatalf("shrunk text %q", d.SelectionText())
	}

	d.SelectNone()
	if _, _, ok := d.Selection(); ok {
		t.Fatal("SelectNone must drop the selection")
	}
	if got := d.SelectionText(); got != "" {
		t.Fatalf("SelectNone text %q", got)
	}
}

func TestMoveWithSelection(t *testing.T) {
	d := newDoc("abc\ndefghi")
	d.SetCursor(Pos{Line: 0, Col: 1})
	d.Move(MoveRight, true) // anchor {0,1}
	// SelectTo leaves desiredCol untouched, so the shift-down aims at the
	// column the cursor came from.
	d.Move(MoveDown, true)
	start, end, ok := d.Selection()
	if !ok || start != (Pos{Line: 0, Col: 1}) || end != (Pos{Line: 1, Col: 1}) {
		t.Fatalf("shift-down selection %v..%v ok=%v", start, end, ok)
	}
	if got := d.SelectionText(); got != "bc\nd" {
		t.Fatalf("extended text %q", got)
	}
	// Further shift-moves keep the original anchor.
	d.Move(MoveRight, true)
	start, _, ok = d.Selection()
	if !ok || start != (Pos{Line: 0, Col: 1}) {
		t.Fatalf("anchor moved: %v ok=%v", start, ok)
	}
}

func TestSelectAllBoundaries(t *testing.T) {
	d := newDoc("abc\ndef\n")
	d.SetCursor(Pos{Line: 1, Col: 1})
	d.SelectAll()
	start, end, ok := d.Selection()
	if !ok || start != (Pos{Line: 0, Col: 0}) || end != (Pos{Line: 2, Col: 0}) {
		t.Fatalf("SelectAll %v..%v ok=%v", start, end, ok)
	}
	if got, want := d.SelectionText(), "abc\ndef\n"; got != want {
		t.Fatalf("SelectAll text %q want %q", got, want)
	}
	s, e, ok := d.SelByteOffsets()
	if !ok || s != 0 || e != len("abc\ndef\n") {
		t.Fatalf("SelByteOffsets %d..%d ok=%v", s, e, ok)
	}

	// Even an empty document selects "nothing" without panicking.
	e2 := newDoc("")
	e2.SelectAll()
	if got := e2.SelectionText(); got != "" {
		t.Fatalf("SelectAll on empty doc %q", got)
	}
	s2, e2n, ok := e2.SelByteOffsets()
	if !ok || s2 != 0 || e2n != 0 {
		t.Fatalf("empty SelectAll offsets %d..%d", s2, e2n)
	}
	if _, _, ok := New().Selection(); ok {
		t.Fatal("fresh document has no selection")
	}
}

func TestSelByteOffsets(t *testing.T) {
	d := newDoc("abc\ndef")
	if _, _, ok := d.SelByteOffsets(); ok {
		t.Fatal("no selection, no offsets")
	}
	d.SetCursor(Pos{Line: 0, Col: 1})
	d.SelectTo(Pos{Line: 1, Col: 2})
	s, e, ok := d.SelByteOffsets()
	// {1,2} in "abc\ndef" is byte offset 4+2 = 6.
	if !ok || s != 1 || e != 6 {
		t.Fatalf("SelByteOffsets %d..%d ok=%v", s, e, ok)
	}
}

func TestInsertRune(t *testing.T) {
	d := newDoc("")
	d.InsertRune('x')
	d.InsertRune('y')
	if got := d.Text(); got != "xy" {
		t.Fatalf("typed %q", got)
	}
	mustPos(t, d.Cursor(), Pos{Line: 0, Col: 2}, "typing advances the cursor")
	if !d.Modified() {
		t.Fatal("typing marks the document modified")
	}

	// Inserting into a selection replaces it.
	r := newDoc("hello world")
	r.SetCursor(Pos{Line: 0, Col: 6})
	r.SelectTo(Pos{Line: 0, Col: 11})
	r.InsertRune('X')
	if got := r.Text(); got != "hello X" {
		t.Fatalf("replace-on-type %q", got)
	}
	mustPos(t, r.Cursor(), Pos{Line: 0, Col: 7}, "cursor after replace-on-type")

	// Unicode rune lands rune-column-correctly.
	u := newDoc("aé")
	u.SetCursor(Pos{Line: 0, Col: 2})
	u.InsertRune('君')
	if got := u.Text(); got != "aé君" {
		t.Fatalf("unicode insert %q", got)
	}
	mustPos(t, u.Cursor(), Pos{Line: 0, Col: 3}, "cursor after unicode insert")
}

func TestInsertTextAndNewline(t *testing.T) {
	d := newDoc("hello world")
	d.SetCursor(Pos{Line: 0, Col: 6})
	d.SelectTo(Pos{Line: 0, Col: 11})
	d.InsertText("panda")
	if got := d.Text(); got != "hello panda" {
		t.Fatalf("InsertText %q", got)
	}
	mustPos(t, d.Cursor(), Pos{Line: 0, Col: 11}, "cursor past inserted text")

	// Newline repeats leading whitespace.
	n := newDoc("    indented")
	n.SetCursor(Pos{Line: 0, Col: 12})
	n.InsertNewline()
	if got := n.Text(); got != "    indented\n    " {
		t.Fatalf("InsertNewline %q", got)
	}
	mustPos(t, n.Cursor(), Pos{Line: 1, Col: 4}, "cursor on fresh indented line")

	// Newline inside a line splits it.
	m := newDoc("foobar")
	m.SetCursor(Pos{Line: 0, Col: 3})
	m.InsertNewline()
	if got := m.Text(); got != "foo\nbar" {
		t.Fatalf("split %q", got)
	}
	mustPos(t, m.Cursor(), Pos{Line: 1, Col: 0}, "cursor after split")
}

func TestBackspace(t *testing.T) {
	d := newDoc("ab\ncd")
	d.SetCursor(Pos{Line: 1, Col: 1})
	d.Backspace() // removes 'c'
	if got := d.Text(); got != "ab\nd" {
		t.Fatalf("backspace %q", got)
	}
	mustPos(t, d.Cursor(), Pos{Line: 1, Col: 0}, "cursor after backspace delete")

	// Joining lines at column 0.
	j := newDoc("ab\ncd")
	j.SetCursor(Pos{Line: 1, Col: 0})
	j.Backspace()
	if got := j.Text(); got != "abcd" {
		t.Fatalf("join %q", got)
	}
	mustPos(t, j.Cursor(), Pos{Line: 0, Col: 2}, "cursor after line join")

	// Backspace at {0,0} is a no-op.
	j.SetCursor(Pos{Line: 0, Col: 0})
	j.Backspace()
	if got := j.Text(); got != "abcd" {
		t.Fatalf("backspace at doc start %q", got)
	}
	mustPos(t, j.Cursor(), Pos{Line: 0, Col: 0}, "cursor untouched at doc start")

	// Backspace over a multi-byte rune.
	u := newDoc("aé")
	u.SetCursor(Pos{Line: 0, Col: 2})
	u.Backspace()
	if got := u.Text(); got != "a" {
		t.Fatalf("multibyte backspace %q", got)
	}
	mustPos(t, u.Cursor(), Pos{Line: 0, Col: 1}, "cursor after multibyte backspace")

	// Selection delete.
	s := newDoc("aXYb")
	s.SetCursor(Pos{Line: 0, Col: 1})
	s.SelectTo(Pos{Line: 0, Col: 3})
	s.Backspace()
	if got := s.Text(); got != "ab" {
		t.Fatalf("selection backspace %q", got)
	}
	mustPos(t, s.Cursor(), Pos{Line: 0, Col: 1}, "cursor at selection start")
}

func TestDeleteForward(t *testing.T) {
	d := newDoc("ab\ncd")
	d.SetCursor(Pos{Line: 0, Col: 2})
	d.DeleteForward() // removes '\n' => joins
	if got := d.Text(); got != "abcd" {
		t.Fatalf("join forward %q", got)
	}
	mustPos(t, d.Cursor(), Pos{Line: 0, Col: 2}, "cursor holds after join")

	// Deleting one rune forward inside a line.
	d2 := newDoc("ab")
	d2.SetCursor(Pos{Line: 0, Col: 1})
	d2.DeleteForward()
	if got := d2.Text(); got != "a" {
		t.Fatalf("delete forward rune %q", got)
	}
	mustPos(t, d2.Cursor(), Pos{Line: 0, Col: 1}, "cursor keeps its spot")

	// Forward over a multi-byte rune.
	u := newDoc("aéb")
	u.SetCursor(Pos{Line: 0, Col: 0})
	u.DeleteForward()
	if got := u.Text(); got != "éb" {
		t.Fatalf("multibyte delete forward %q", got)
	}
	mustPos(t, u.Cursor(), Pos{Line: 0, Col: 0}, "cursor holds after rune delete")

	// End of the document is a no-op.
	e := newDoc("a")
	e.SetCursor(Pos{Line: 0, Col: 1})
	e.DeleteForward()
	if got := e.Text(); got != "a" {
		t.Fatalf("delete at doc end %q", got)
	}

	// Selection delete.
	s := newDoc("aXYb")
	s.SetCursor(Pos{Line: 0, Col: 0})
	s.SelectTo(Pos{Line: 0, Col: 3})
	s.DeleteForward()
	if got := s.Text(); got != "b" || s.Cursor() != (Pos{Line: 0, Col: 0}) {
		t.Fatalf("selection delete forward %q cursor %v", got, s.Cursor())
	}
}

func TestDeleteWord(t *testing.T) {
	d := newDoc("foo bar baz")
	d.SetCursor(Pos{Line: 0, Col: 7})
	// From col 7 the backward scan removes the adjacent "bar" run only.
	d.DeleteWordBackward()
	if got := d.Text(); got != "foo  baz" {
		t.Fatalf("DeleteWordBackward %q", got)
	}
	mustPos(t, d.Cursor(), Pos{Line: 0, Col: 4}, "cursor after word delete")

	// When the cursor sits inside the gap run, the scan spans it and then
	// the preceding word: "alpha beta" -> "beta".
	g := newDoc("alpha beta")
	g.SetCursor(Pos{Line: 0, Col: 6})
	g.DeleteWordBackward()
	if got := g.Text(); got != "beta" {
		t.Fatalf("DeleteWordBackward across space %q", got)
	}
	mustPos(t, g.Cursor(), Pos{Line: 0, Col: 0}, "cursor at collapsed start")

	f := newDoc("foo bar baz")
	f.SetCursor(Pos{Line: 0, Col: 7})
	f.DeleteWordForward() // removes " baz"
	if got := f.Text(); got != "foo bar" {
		t.Fatalf("DeleteWordForward %q", got)
	}
	mustPos(t, f.Cursor(), Pos{Line: 0, Col: 7}, "cursor keeps its spot after forward word delete")

	// Line edges stop the scan: no joining across newlines.
	l := newDoc("foo\nbar")
	l.SetCursor(Pos{Line: 0, Col: 3})
	l.DeleteWordForward()
	if got := l.Text(); got != "foo\nbar" {
		t.Fatalf("forward at line end must not join: %q", got)
	}
	l.SetCursor(Pos{Line: 1, Col: 0})
	l.DeleteWordBackward()
	if got := l.Text(); got != "foo\nbar" {
		t.Fatalf("backward at column 0 must not join: %q", got)
	}

	// Multi-byte word support.
	u := newDoc("漢字word rest")
	u.SetCursor(Pos{Line: 0, Col: 6})
	u.DeleteWordBackward()
	if got := u.Text(); got != " rest" {
		t.Fatalf("unicode word delete %q", got)
	}
	// Forward scan spans the gap, then the word run.
	u.SetCursor(Pos{Line: 0, Col: 0})
	u.DeleteWordForward()
	if got := u.Text(); got != "" {
		t.Fatalf("unicode word delete forward %q", got)
	}
}

func TestIndentDedentMultiline(t *testing.T) {
	d := newDoc("  foo=1\n  bar\t2\n  baz\nqux\n")
	d.SetCursor(Pos{Line: 1, Col: 2})
	d.SelectTo(Pos{Line: 3, Col: 0}) // selection touches lines 1 and 2 (caret on col 0 excluded)
	d.Indent()
	want := "  foo=1\n\t  bar\t2\n\t  baz\nqux\n"
	if got := d.Text(); got != want {
		t.Fatalf("Indent: %q want %q", got, want)
	}
	// The caret stayed at the selection endpoint; resettle only re-clamps.
	mustPos(t, d.Cursor(), Pos{Line: 3, Col: 0}, "cursor after Indent")

	// Round trip through Undo.
	if !d.Undo() {
		t.Fatal("Indent must be undoable")
	}
	if got := d.Text(); got != "  foo=1\n  bar\t2\n  baz\nqux\n" {
		t.Fatalf("Indent undo %q", got)
	}
	if !d.Redo() {
		t.Fatal("Indent must be redoable")
	}
	if got := d.Text(); got != want {
		t.Fatalf("Indent redo %q", got)
	}

	// Dedent takes one leading tab per touched line.
	d.SetCursor(Pos{Line: 1, Col: 2})
	d.SelectTo(Pos{Line: 4, Col: 0}) // touches lines 1, 2, 3 ("qux" has no indent)
	d.Dedent()
	want2 := "  foo=1\n  bar\t2\n  baz\nqux\n"
	if got := d.Text(); got != want2 {
		t.Fatalf("Dedent: %q want %q", got, want2)
	}

	// No selection: Indent inserts a tab at the cursor.
	single := newDoc("line\nnext")
	single.SetCursor(Pos{Line: 0, Col: 0})
	single.Indent()
	if got := single.Text(); got != "\tline\nnext" {
		t.Fatalf("single-line Indent %q", got)
	}
	mustPos(t, single.Cursor(), Pos{Line: 0, Col: 1}, "cursor after single-line Indent")

	// No selection: Dedent strips the cursor's line.
	single.Dedent()
	if got := single.Text(); got != "line\nnext" {
		t.Fatalf("single-line Dedent %q", got)
	}

	// Space collapse: four spaces become one tab unit.
	sp := newDoc("    a\n  b\n\tc\n  d")
	sp.SelectNone()
	sp.SetCursor(Pos{Line: 0, Col: 0})
	sp.SelectTo(Pos{Line: 3, Col: 3}) // touches lines 0..3
	sp.Dedent()
	want3 := "a\nb\nc\nd"
	if got := sp.Text(); got != want3 {
		t.Fatalf("space collapse %q want %q", got, want3)
	}
}

func TestScrollClampAndEnsureCursorVisible(t *testing.T) {
	d := newDoc("a\nb\nc\nd\ne\nf") // 6 lines
	d.SetScrollY(99)
	if got := d.ScrollY(); got != 5 {
		t.Fatalf("SetScrollY clamp high: %d", got)
	}
	d.SetScrollY(-3)
	if got := d.ScrollY(); got != 0 {
		t.Fatalf("SetScrollY clamp low: %d", got)
	}

	d.SetCursor(Pos{Line: 3, Col: 0})
	d.SetScrollY(5)
	d.EnsureCursorVisible(3) // cursor above the window
	if got := d.ScrollY(); got != 3 {
		t.Fatalf("EnsureCursorVisible up: %d", got)
	}
	d.SetCursor(Pos{Line: 5, Col: 0})
	d.SetScrollY(0)
	d.EnsureCursorVisible(2) // cursor below the window
	if got := d.ScrollY(); got != 4 {
		t.Fatalf("EnsureCursorVisible down: %d", got)
	}
	// Already visible: no change.
	d.SetCursor(Pos{Line: 4, Col: 0})
	d.EnsureCursorVisible(2)
	if got := d.ScrollY(); got != 4 {
		t.Fatalf("EnsureCursorVisible stable: %d", got)
	}
}

func TestReadOnlyGuard(t *testing.T) {
	d := newDoc("abc\nxyz")
	d.SetCursor(Pos{Line: 0, Col: 1})
	d.SelectTo(Pos{Line: 1, Col: 2})
	d.SetReadOnly(true)

	d.InsertRune('Q')
	d.InsertText("boom")
	d.InsertNewline()
	d.Backspace()
	d.DeleteForward()
	d.DeleteWordBackward()
	d.DeleteWordForward()
	d.Indent()
	d.Dedent()

	if got := d.Text(); got != "abc\nxyz" {
		t.Fatalf("read-only edits changed the buffer: %q", got)
	}
	// Cursor and selection survive the refused edits.
	start, end, ok := d.Selection()
	if !ok || start != (Pos{Line: 0, Col: 1}) || end != (Pos{Line: 1, Col: 2}) {
		t.Fatalf("selection survived? %v..%v ok=%v", start, end, ok)
	}
	if d.Undo() {
		t.Fatal("Undo refused when read-only")
	}
	if d.Redo() {
		t.Fatal("Redo refused when read-only")
	}

	// Edits work again once the flag is dropped (no selection: DeleteForward
	// cuts just one rune rather than the stale endpoints save).
	d.SetReadOnly(false)
	d.SelectNone()
	d.SetCursor(Pos{Line: 0, Col: 0})
	d.DeleteForward()
	if got := d.Text(); got != "bc\nxyz" {
		t.Fatalf("unlocked edit: %q", got)
	}
}

func TestUndoRedoRoundTrip(t *testing.T) {
	orig := "ab\ncd\n"
	d := newDoc(orig)
	d.SetCursor(Pos{Line: 0, Col: 0})
	d.InsertRune('x')
	d.InsertRune('y') // typing-burst coalesces: one step
	d.ReplaceRange(1, 2, "MN")

	if !d.Undo() {
		t.Fatal("first undo failed")
	}
	if !d.Undo() {
		t.Fatal("second undo failed")
	}
	if got := d.Text(); got != orig {
		t.Fatalf("text after two undos: %q", got)
	}
	// ReplaceRange left the cursor right after the replacement; undoing it
	// clamps that position back inside the shrunk buffer.
	mustPos(t, d.Cursor(), Pos{Line: 0, Col: 2}, "cursor clamped on undo")

	if !d.Redo() {
		t.Fatal("first redo failed")
	}
	if got := d.Text(); got != "xyab\ncd\n" {
		t.Fatalf("first redo %q", got)
	}
	if !d.Redo() {
		t.Fatal("second redo failed")
	}
	// Redo never restores a remembered cursor: the current cursor is only
	// re-clamped after each step.
	mustPos(t, d.Cursor(), Pos{Line: 0, Col: 2}, "cursor held on redo")

	// A fresh edit clears the redo tail: an exhausted Undo reports false.
	d.InsertRune('!')
	if d.Redo() {
		t.Fatal("redo must be gone after a fresh edit")
	}

	// Three steps made it back to the start: the '!' burst plus the two
	// original steps.
	if !d.Undo() || !d.Undo() || !d.Undo() {
		t.Fatal("undo chain must reply thrice")
	}
	if got := d.Text(); got != orig {
		t.Fatalf("final undo %q", got)
	}
	if d.Undo() {
		t.Fatal("Undo past the beginning reports false")
	}

	// A line join reverts cleanly too.
	j := newDoc("ab\ncd")
	j.SetCursor(Pos{Line: 1, Col: 0})
	j.Backspace()
	if got := j.Text(); got != "abcd" {
		t.Fatalf("join: %q", got)
	}
	if !j.Undo() {
		t.Fatal("join undo failed")
	}
	if got := j.Text(); got != "ab\ncd" {
		t.Fatalf("join undo text %q", got)
	}
	if !j.Redo() {
		t.Fatal("join redo failed")
	}
	if got := j.Text(); got != "abcd" {
		t.Fatalf("join redo text %q", got)
	}
	// The join was one step; exhausting the stack reports false.
	if !j.Undo() {
		t.Fatal("exhaustive undo failed")
	}
	if j.Undo() {
		t.Fatal("exhausted undo reports false")
	}
}

func TestReplaceRange(t *testing.T) {
	d := newDoc("hello world")
	off := d.ReplaceRange(6, 11, "panda")
	if off != 11 {
		t.Fatalf("ReplaceRange returned %d", off)
	}
	if got := d.Text(); got != "hello panda" {
		t.Fatalf("ReplaceRange text %q", got)
	}
	mustPos(t, d.Cursor(), Pos{Line: 0, Col: 11}, "cursor just past the replacement")
	if _, _, ok := d.Selection(); ok {
		t.Fatal("ReplaceRange must not leave a selection")
	}

	// Replacing with a longer string shifts the rest, then shrinks again.
	up := d.ReplaceRange(6, 11, "editor")
	if up != 12 || d.Text() != "hello editor" {
		t.Fatalf("grow: %q off=%d", d.Text(), up)
	}
	back := d.ReplaceRange(6, 12, "p")
	if back != 7 || d.Text() != "hello p" {
		t.Fatalf("shrink: %q off=%d", d.Text(), back)
	}

	// Newline insertions move the cursor onto the new line.
	n := newDoc("abcd")
	got := n.ReplaceRange(2, 2, "\n")
	if got != 3 || n.Text() != "ab\ncd" {
		t.Fatalf("newline replace: %q off=%d", n.Text(), got)
	}
	mustPos(t, n.Cursor(), Pos{Line: 1, Col: 0}, "cursor on the fresh line")

	// Out-of-range arguments clamp instead of panicking: both ends land
	// inside the buffer, so only [0,3) is replaced.
	c := newDoc("abc")
	if off := c.ReplaceRange(-5, 99, "z"); off != 1 || c.Text() != "z" {
		t.Fatalf("clamped range: %q off=%d", c.Text(), off)
	}
	mustPos(t, c.Cursor(), Pos{Line: 0, Col: 1}, "cursor after clamped replace")

	// Reversed range is normalized.
	r := newDoc("abcdef")
	if off := r.ReplaceRange(4, 1, "X"); off != 2 || r.Text() != "aXef" {
		t.Fatalf("reversed range: %q off=%d", r.Text(), off)
	}

	// ReplaceRange is one undo step per call; three calls undo back.
	d.SetCursor(Pos{Line: 0, Col: 0})
	if !d.Undo() || !d.Undo() || !d.Undo() {
		t.Fatal("ReplaceRange undo failed")
	}
	if got := d.Text(); got != "hello world" {
		t.Fatalf("ReplaceRange undo chain: %q", got)
	}

	// Empty replacement on an empty buffer is a well-behaved no-op.
	e := newDoc("")
	if off := e.ReplaceRange(0, 0, ""); off != 0 || e.Text() != "" {
		t.Fatalf("empty replace: %q off=%d", e.Text(), off)
	}
}

func TestGotoLine(t *testing.T) {
	d := newDoc("one\ntwo\nthree")
	d.GotoLine(2)
	mustPos(t, d.Cursor(), Pos{Line: 2, Col: 0}, "GotoLine")
	d.GotoLine(99)
	mustPos(t, d.Cursor(), Pos{Line: 2, Col: 0}, "GotoLine clamps")
	d.GotoLine(-4)
	mustPos(t, d.Cursor(), Pos{Line: 0, Col: 0}, "GotoLine clamps low")
}
