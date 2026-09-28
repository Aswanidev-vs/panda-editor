package editorview

import (
	"strings"
	"testing"

	"github.com/Aswanidev-vs/cherry/cell"
	"github.com/Aswanidev-vs/cherry/geom"
	"github.com/Aswanidev-vs/cherry/input"
	"github.com/Aswanidev-vs/cherry/render"
	"github.com/Aswanidev-vs/cherry/widget"

	"github.com/Aswanidev-vs/panda-editor/editor/document"
	"github.com/Aswanidev-vs/panda-editor/editor/theme"
)

const textX = 3 // gutter(2) + margin(1) at the rect origin for line counts < 10

func textOriginForTest(lineCount int) int {
	return textOrigin(0, numWidth(lineCount))
}

type harness struct {
	v   *View
	doc *document.Document
	scr *render.Screen
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	doc := document.New()
	v := New(nil, doc)
	v.Focus()
	scr := render.New(80, 24)
	size := v.Measure(geom.Size{W: 80, H: 24})
	if size.W != 80 || size.H != 24 {
		t.Fatalf("Measure: got %+v, want fill 80x24", size)
	}
	v.Draw(&widget.DrawCtx{Rect: geom.Rect{Size: size}, Screen: scr})
	return &harness{v: v, doc: doc, scr: scr}
}

func (h *harness) draw() {
	h.scr.Clear()
	h.v.Draw(&widget.DrawCtx{Rect: geom.Rect{Size: h.scr.Size()}, Screen: h.scr})
}

func (h *harness) key(k input.Key, m ...input.Mod) bool {
	mod := input.Mod(0)
	for _, x := range m {
		mod |= x
	}
	return h.v.Handle(input.KeyPress{Key: k, Mod: mod})
}

func (h *harness) runeKey(r rune, m ...input.Mod) bool {
	mod := input.Mod(0)
	for _, x := range m {
		mod |= x
	}
	return h.v.Handle(input.KeyPress{Rune: r, Mod: mod})
}

func lineText(scr *render.Screen, y, x0, n int) string {
	runes := make([]rune, 0, n)
	for x := x0; x < x0+n; x++ {
		c := scr.CellAt(x, y)
		if c.Rune == 0 {
			break
		}
		runes = append(runes, c.Rune)
	}
	return strings.TrimRight(string(runes), " ")
}

func typeText(v *View, s string) {
	for _, r := range s {
		if r == '\n' {
			v.Handle(input.KeyPress{Key: input.KeyEnter})
			continue
		}
		v.Handle(input.KeyPress{Rune: r})
	}
}

func TestGutterLineNumbers(t *testing.T) {
	h := newHarness(t)
	typeText(h.v, "hello\nworld")
	h.draw()

	// 2 lines -> nw=3. Geometry: rail(0), num(1..3), tick(4), blanks(5,6), text(7)
	// Number "1" right-aligned in 3-wide field at x=3.
	if got := lineText(h.scr, 0, 3, 1); got != "1" {
		t.Errorf("row0 gutter number = %q at x=3, want %q", got, "1")
	}
	if got := lineText(h.scr, 1, 3, 1); got != "2" {
		t.Errorf("row1 gutter number = %q at x=3, want %q", got, "2")
	}
	// Tick rule is TickLight (┆) at x=4 on non-cursor lines.
	if c := h.scr.CellAt(4, 0); c.Rune != theme.Glyph.TickLight {
		t.Errorf("gutter tick rule = %q, want %q (TickLight)", c.Rune, theme.Glyph.TickLight)
	}
	// The tick rule becomes TickHeavy (│) on the cursor line. After typing
	// "hello\nworld" the cursor sits on line 1, so the cursor line is row 1 —
	// not row 0, which is why this needs its own row to be meaningful.
	h.v.Focus()
	h.draw()
	if c := h.scr.CellAt(4, 1); c.Rune != theme.Glyph.TickHeavy {
		t.Errorf("cursor-line tick rule = %q, want %q (TickHeavy)", c.Rune, theme.Glyph.TickHeavy)
	}
	if c := h.scr.CellAt(0, 1); c.Rune != theme.Glyph.BlockLeft {
		t.Errorf("cursor-line rail marker = %q, want %q (BlockLeft)", c.Rune, theme.Glyph.BlockLeft)
	}
	// The cursor-line wash spans the whole row, so the gutter cells on the
	// cursor line carry the cursor-line background, not the sunken one.
	if c := h.scr.CellAt(3, 1); c.Style.Bg != theme.Current.CursorLine {
		t.Errorf("cursor-line gutter background = %#v, want the cursor-line wash", c.Style.Bg)
	}
	if c := h.scr.CellAt(3, 0); c.Style.Bg != theme.Current.Sunken {
		t.Errorf("idle-line gutter background = %#v, want the sunken well", c.Style.Bg)
	}
	// Line-number cell at x=3 has truecolour foreground.
	if c := h.scr.CellAt(3, 0); !c.Style.Fg.IsRGB() {
		t.Errorf("line-number cell style = %+v, want truecolour foreground", c.Style)
	}
	// Virtual-area gutter cells (rail + number + tick) are blank / tick.
	if c := h.scr.CellAt(0, 2); c.Rune != ' ' {
		t.Errorf("virtual-area rail cell = %+v, want blank", c)
	}
	if c := h.scr.CellAt(3, 2); c.Rune != ' ' {
		t.Errorf("virtual-area number cell = %+v, want blank", c)
	}
	if c := h.scr.CellAt(4, 2); c.Rune != theme.Glyph.TickLight {
		t.Errorf("virtual-area tick cell = %q, want %q", c.Rune, theme.Glyph.TickLight)
	}
}

func TestTextSpansDraw(t *testing.T) {
	h := newHarness(t)
	typeText(h.v, "hello\nworld")
	h.draw()

	tx := textOriginForTest(2)
	if got := lineText(h.scr, 0, tx, 10); got != "hello" {
		t.Errorf("line0 = %q, want hello", got)
	}
	if got := lineText(h.scr, 1, tx, 10); got != "world" {
		t.Errorf("line1 = %q, want world", got)
	}
	if c := h.scr.CellAt(tx+5, 0); c.Rune != ' ' || !c.Style.Bg.IsRGB() {
		t.Errorf("cell past EOL = %+v, want theme background", c)
	}
}

func TestNoLineWrapping(t *testing.T) {
	h := newHarness(t)
	long := strings.Repeat("a", 120)
	typeText(h.v, long)
	h.draw()

	tx := textOriginForTest(1)
	want := long[:80-tx]
	if got := lineText(h.scr, 0, tx, 80-tx); got != want {
		t.Errorf("row0 drawn %d chars, want %d (clipped at right edge)", len(got), len(want))
	}
	// Past EOF shows MiddleDot (·) at text origin, not ~.
	if got := lineText(h.scr, 1, tx, 1); got != string(theme.Glyph.MiddleDot) {
		t.Errorf("row1 = %q, want %q virtual-line marker", got, string(theme.Glyph.MiddleDot))
	}
	if lc := h.doc.LineCount(); lc != 1 {
		t.Errorf("LineCount = %d, want 1 (wrapping disabled)", lc)
	}
}

func TestTabExpansion(t *testing.T) {
	h := newHarness(t)
	// insert a literal tab rune so the test is independent of Indent's
	// tab-vs-spaces choice
	if !h.runeKey('\t') {
		t.Fatal("tab rune not consumed")
	}
	typeText(h.v, "x")
	h.draw()

	tx := textOriginForTest(1)
	if got := lineText(h.scr, 0, tx, 10); got != "    x" {
		t.Errorf("row0 = %q, want tab expanded to 4 spaces", got)
	}
	h.v.SetTabWidth(2)
	h.draw()
	if got := lineText(h.scr, 0, tx, 10); got != "  x" {
		t.Errorf("row0 after SetTabWidth(2) = %q, want 2-space tab", got)
	}
}

func TestSelectionBackground(t *testing.T) {
	h := newHarness(t)
	typeText(h.v, "hello\nworld")
	if !h.key(input.KeyHome, input.ModCtrl) {
		t.Fatal("ctrl+home not consumed")
	}
	for i := 0; i < 5; i++ {
		if !h.key(input.KeyRight, input.ModShift) {
			t.Fatal("shift+right not consumed")
		}
	}
	h.draw()

	tx := textOriginForTest(2)
	selBg := theme.Current.Selection
	for x := tx; x < tx+5; x++ {
		c := h.scr.CellAt(x, 0)
		if c.Rune != rune("hello"[x-tx]) {
			t.Errorf("cell %d rune = %q, want %q", x, c.Rune, "hello"[x-tx])
		}
		// Selection uses explicit background, not reverse video.
		if c.Style.Bg != selBg {
			t.Errorf("cell %d bg = %+v, want Selection background %+v", x, c.Style.Bg, selBg)
		}
		if c.Style.Attrs&cell.AttrReverse != 0 {
			t.Errorf("cell %d must not have AttrReverse set", x)
		}
	}
	// Cell past selection end has normal background, not reversed.
	if c := h.scr.CellAt(tx+5, 0); c.Style.Bg == selBg {
		t.Errorf("cell past selection end must not have Selection bg, style %+v", c.Style)
	}
	// Second line is outside the selection.
	if c := h.scr.CellAt(tx, 1); c.Style.Bg == selBg {
		t.Error("line1 first cell has Selection bg, selection is line0 only")
	}
	// Unselected cells on the selected line (before selection start) have normal bg.
	// (Selection starts at col 0 in this test, so nothing before it.)
}

func TestCursorPosColumnCalc(t *testing.T) {
	h := newHarness(t)
	typeText(h.v, "hello\nworld")
	if got := h.doc.Cursor(); got.Line != 1 || got.Col != 5 {
		t.Fatalf("cursor after typing = %+v, want Line1 Col5", got)
	}
	h.key(input.KeyHome, input.ModCtrl)
	h.key(input.KeyRight)
	h.key(input.KeyRight)
	h.draw()

	if got := h.doc.Cursor(); got.Line != 0 || got.Col != 2 {
		t.Fatalf("cursor = %+v, want Line0 Col2", got)
	}
	pos, ok := h.v.CursorPos()
	if !ok {
		t.Fatal("CursorPos: want visible=true")
	}
	if pos.Y != 0 {
		t.Errorf("CursorPos.Y = %d, want 0", pos.Y)
	}
	// x = rect.X + rail(1) + numWidth(3) + tick(1) + blanks(2) + width("he")
	//     = 0 + 1 + 3 + 1 + 2 + 2 = 9
	tx := textOriginForTest(2)
	if pos.X != tx+2 {
		t.Fatalf("CursorPos.X = %d, want %d (textOrigin + rune widths of prefix)", pos.X, tx+2)
	}

	h.v.Blur()
	if _, ok := h.v.CursorPos(); ok {
		t.Error("CursorPos after Blur must be hidden")
	}
}

func TestCursorPosWideAndTab(t *testing.T) {
	h := newHarness(t)
	typeText(h.v, "\t你好")
	h.draw()

	if got := h.doc.Cursor(); got.Col != 3 {
		t.Fatalf("cursor col = %d, want 3 (tab + 2 wide runes)", got.Col)
	}
	pos, ok := h.v.CursorPos()
	if !ok {
		t.Fatal("CursorPos: want visible=true")
	}
	// tab 4 + two wide runes (4 cols) after the text origin
	tx := textOriginForTest(1)
	if pos.X != tx+8 || pos.Y != 0 {
		t.Errorf("CursorPos = %+v, want {%d 0}", pos, tx+8)
	}
}

func TestScrollRepositionAfterMove(t *testing.T) {
	h := newHarness(t)
	var b strings.Builder
	for i := 0; i < 40; i++ {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("L" + strings.Repeat("x", i%5))
		b.WriteString(fmtLineTag(i))
	}
	typeText(h.v, b.String())
	if lc := h.doc.LineCount(); lc != 40 {
		t.Fatalf("LineCount = %d, want 40", lc)
	}

	// jump far down: scroll must follow so the cursor row lands in view
	h.doc.SetCursor(document.Pos{Line: 36, Col: 0})
	h.v.updateScroll()
	h.draw()

	if sy := h.doc.ScrollY(); sy == 0 {
		t.Fatal("ScrollY must have moved after jumping to line 36")
	}
	if sy := h.doc.ScrollY(); sy < 36-24+1 {
		t.Errorf("ScrollY = %d, want >= %d so line 36 stays visible", sy, 36-24+1)
	}
	// 40 lines -> digitCount=2, nw=3. Number field at x=1..3, right-aligned.
	// Topmost drawn line number is ScrollY+1, at x=3 (rightmost of 3-wide field).
	got := lineText(h.scr, 0, 3, 1)
	want := rightAlign(h.doc.ScrollY()+1, 3)[2:] // last char of 3-wide right-align
	if got != want {
		t.Errorf("pixel row0 gutter number = %q, want %q (ScrollY offset)", got, want)
	}
	pos, ok := h.v.CursorPos()
	if !ok {
		t.Fatal("cursor for line 36 must be visible after scroll-follow")
	}
	dy := 36 - h.doc.ScrollY()
	if pos.Y != dy {
		t.Errorf("CursorPos.Y = %d, want %d (line - ScrollY)", pos.Y, dy)
	}

	// move within view must not re-scroll
	sy := h.doc.ScrollY()
	h.key(input.KeyUp)
	if h.doc.ScrollY() != sy {
		t.Errorf("ScrollY changed to %d after an in-view move, want %d", h.doc.ScrollY(), sy)
	}

	// page down must keep the cursor visible (repositions scroll)
	h.key(input.KeyPageDown)
	pos, ok = h.v.CursorPos()
	if !ok {
		t.Error("cursor must be visible after PageDown scroll-follow")
	}
	if cur := h.doc.Cursor(); cur.Line-pos.Y != h.doc.ScrollY() {
		t.Errorf("cursor line %d, posY %d, ScrollY %d inconsistent", cur.Line, pos.Y, h.doc.ScrollY())
	}
}

func TestModalIntercepts(t *testing.T) {
	h := newHarness(t)
	calls := 0
	h.v.SetModal(funcModal(func(k input.KeyPress, d *document.Document) bool {
		calls++
		return k.Key == input.KeyEnter
	}))
	if !h.key(input.KeyEnter) {
		t.Fatal("enter must be consumed when the modal claims it")
	}
	if lc := h.doc.LineCount(); lc != 1 {
		t.Errorf("LineCount = %d, modal-consumed enter must not insert newline", lc)
	}
	h.runeKey('a')
	h.v.SetModal(nil)
	if calls != 2 {
		t.Errorf("modal HandleKey calls = %d, want 2", calls)
	}
	if got := h.doc.Buffer().Text(); got != "a" {
		t.Errorf("buffer = %q, want %q (modal let the rune through)", got, "a")
	}
}

func TestPasteAndReadOnlyGuards(t *testing.T) {
	h := newHarness(t)
	if consumed := h.v.Handle(input.Paste{Text: "abc"}); !consumed {
		t.Fatal("paste must be consumed")
	}
	if got := h.doc.Buffer().Text(); got != "abc" {
		t.Fatalf("buffer after paste = %q, want abc", got)
	}

	// resize is reported, not consumed
	if consumed := h.v.Handle(input.Resize{Width: 80, Height: 12}); consumed {
		t.Error("resize must return false so other layers can act")
	}

	h.doc.SetReadOnly(true)
	h.runeKey('x')
	if got := h.doc.Buffer().Text(); got != "abc" {
		t.Errorf("read-only typing mutated buffer: %q", got)
	}
	h.doc.SetReadOnly(false)
}

type funcModal func(input.KeyPress, *document.Document) bool

func (f funcModal) HandleKey(k input.KeyPress, d *document.Document) bool { return f(k, d) }

func fmtLineTag(i int) string {
	const digits = "0123456789"
	if i < 10 {
		return string(digits[i])
	}
	return string(digits[i/10]) + string(digits[i%10])
}

func rightAlign(n, w int) string {
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if s == "" {
		s = "0"
	}
	for len(s) < w {
		s = " " + s
	}
	return s[len(s)-w:]
}
