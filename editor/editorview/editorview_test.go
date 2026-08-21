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
)

const textX = 3 // gutter(2) + margin(1) at the rect origin for line counts < 10

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

	if got := lineText(h.scr, 0, 0, 3); got != "1" {
		t.Errorf("row0 gutter = %q, want %q right-aligned in the single label column", got, "1")
	}
	if got := lineText(h.scr, 1, 0, 3); got != "2" {
		t.Errorf("row1 gutter = %q, want %q", got, "2")
	}
	if c := h.scr.CellAt(2, 0); c.Rune != ' ' || c.Style.Fg.IsIndexed() {
		t.Errorf("gutter margin cell = %+v, want blank", c)
	}
	if c := h.scr.CellAt(1, 1); !c.Style.Fg.IsIndexed() {
		t.Errorf("line-number cell style = %+v, want dim foreground", c.Style)
	}
	if c := h.scr.CellAt(0, 2); c.Rune != ' ' {
		t.Errorf("virtual-area gutter cell = %+v, want blank", c)
	}
}

func TestTextSpansDraw(t *testing.T) {
	h := newHarness(t)
	typeText(h.v, "hello\nworld")
	h.draw()

	if got := lineText(h.scr, 0, textX, 10); got != "hello" {
		t.Errorf("line0 = %q, want hello", got)
	}
	if got := lineText(h.scr, 1, textX, 10); got != "world" {
		t.Errorf("line1 = %q, want world", got)
	}
	if c := h.scr.CellAt(textX+5, 0); c.Rune != ' ' || !c.Style.Bg.IsDefault() {
		t.Errorf("cell past EOL = %+v, want unstyled blank", c)
	}
}

func TestNoLineWrapping(t *testing.T) {
	h := newHarness(t)
	long := strings.Repeat("a", 120)
	typeText(h.v, long)
	h.draw()

	want := long[:80-textX]
	if got := lineText(h.scr, 0, textX, 80-textX); got != want {
		t.Errorf("row0 drawn %d chars, want %d (clipped at right edge)", len(got), len(want))
	}
	if got := lineText(h.scr, 1, textX, 10); got != "" {
		t.Errorf("row1 = %q, want empty (horizontal-only, no wrap)", got)
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

	if got := lineText(h.scr, 0, textX, 10); got != "    x" {
		t.Errorf("row0 = %q, want tab expanded to 4 spaces", got)
	}
	h.v.SetTabWidth(2)
	h.draw()
	if got := lineText(h.scr, 0, textX, 10); got != "  x" {
		t.Errorf("row0 after SetTabWidth(2) = %q, want 2-space tab", got)
	}
}

func TestSelectionInversion(t *testing.T) {
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

	for x := textX; x < textX+5; x++ {
		c := h.scr.CellAt(x, 0)
		if c.Rune != rune("hello"[x-textX]) {
			t.Errorf("cell %d rune = %q, want %q", x, c.Rune, "hello"[x-textX])
		}
		if c.Style.Attrs&cell.AttrReverse == 0 {
			t.Errorf("cell %d style %+v: selection must set AttrReverse", x, c.Style)
		}
	}
	if c := h.scr.CellAt(textX+5, 0); c.Style.Attrs&cell.AttrReverse != 0 {
		t.Errorf("cell past selection end must not be reversed, style %+v", c.Style)
	}
	// the second line is outside the selection
	if c := h.scr.CellAt(textX, 1); c.Style.Attrs&cell.AttrReverse != 0 {
		t.Error("line1 first cell reversed, selection is line0 only")
	}
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
	// x = rect.X + gutter(2) + margin(1) + width("he") = 0 + 2 + 1 + 2
	if pos.X != textX+2 {
		t.Fatalf("CursorPos.X = %d, want %d (gutter+margin + rune widths of prefix)", pos.X, textX+2)
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
	// tab 4 + two wide runes (4 cols) after the gutter/margin offset
	if pos.X != textX+8 || pos.Y != 0 {
		t.Errorf("CursorPos = %+v, want {%d 0}", pos, textX+8)
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
	// topmost drawn line number is ScrollY+1; 40 lines -> two label columns
	// plus a blank margin column, so a 2-digit number fills cols 0..1
	got := lineText(h.scr, 0, 0, 3)
	want := rightAlign(h.doc.ScrollY()+1, 2)
	if got != want {
		t.Errorf("pixel row0 gutter = %q, want %q (ScrollY offset)", got, want)
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
