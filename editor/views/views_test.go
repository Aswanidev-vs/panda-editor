package views

import (
	"strings"
	"testing"

	"github.com/Aswanidev-vs/cherry/geom"
	"github.com/Aswanidev-vs/cherry/input"
	"github.com/Aswanidev-vs/cherry/render"
	"github.com/Aswanidev-vs/cherry/widget"
)

func key(r rune) input.KeyPress     { return input.KeyPress{Key: input.KeyNone, Rune: r} }
func k(kk input.Key) input.KeyPress { return input.KeyPress{Key: kk} }

// drawAt paints w into rect r of a fresh 80x24 screen and returns it.
func drawAt(t *testing.T, w widget.Widget, r geom.Rect) *render.Screen {
	t.Helper()
	sc := render.New(80, 24)
	w.Draw(&widget.DrawCtx{Rect: r, Screen: sc})
	return sc
}

// rowText concatenates the runes of row y across x..right.
func rowText(sc *render.Screen, y, x, right int) string {
	var b strings.Builder
	for ; x < right; x++ {
		b.WriteRune(sc.CellAt(x, y).Rune)
	}
	return b.String()
}

func TestStatusBarText(t *testing.T) {
	segs := StatusBarText("insert", "main.go", "saved", 3, 12, false, false)
	if len(segs) != 3 {
		t.Fatalf("segments = %d, want 3", len(segs))
	}
	if !strings.Contains(segs[0].Text, "INSERT") {
		t.Errorf("mode missing from first segment %q", segs[0].Text)
	}
	if !strings.Contains(segs[0].Text, "main.go") {
		t.Errorf("file missing from first segment %q", segs[0].Text)
	}
	if strings.Contains(segs[0].Text, "+") {
		t.Errorf("unexpected modified marker in %q", segs[0].Text)
	}
	if !strings.Contains(segs[1].Text, "saved") {
		t.Errorf("message missing from middle segment %q", segs[1].Text)
	}
	if !strings.Contains(segs[2].Text, "Ln 3, Col 12") {
		t.Errorf("position missing from last segment %q", segs[2].Text)
	}

	mod := StatusBarText("edit", "notes.md", "", 1, 1, true, true)
	if !strings.Contains(mod[0].Text, "+") {
		t.Errorf("modified marker missing in %q", mod[0].Text)
	}
	if !strings.Contains(mod[0].Text, "[RO]") {
		t.Errorf("readonly marker missing in %q", mod[0].Text)
	}

	// Empty inputs still yield three usable segments.
	empty := StatusBarText("", "", "", 0, 0, false, false)
	if len(empty) != 3 {
		t.Fatalf("empty segments = %d, want 3", len(empty))
	}
}

func TestHintBar(t *testing.T) {
	for _, mode := range []string{"insert", "search", "dialog", "welcome"} {
		hb := HintBar(mode)
		if hb == nil || hb.Content == "" {
			t.Fatalf("HintBar(%s) produced no content", mode)
		}
		if hb.Measure(geom.Size{}).H != 1 {
			t.Errorf("HintBar(%s) must be a single row", mode)
		}
	}
	if HintBar("unknown").Content == "" {
		t.Error("unknown mode must fall back to a generic strip")
	}
}

func TestInputLineTypingBackspaceAndDraw(t *testing.T) {
	var ok string
	line := New("Name: ", "hi", func(s string) { ok = s }, func() {})
	line.Focus()
	if !line.Focused() {
		t.Fatal("Focus did not set focused state")
	}

	if got := line.Handle(key('!')); !got {
		t.Fatal("rune keypress must be consumed")
	}
	line.Handle(key('A'))
	if line.Text() != "hi!A" || line.CursorCol() != 4 {
		t.Fatalf("type at end: text %q cursor %d", line.Text(), line.CursorCol())
	}

	line.Handle(k(input.KeyHome))
	line.Handle(key('x'))
	if line.Text() != "xhi!A" {
		t.Fatalf("insert at home: %q", line.Text())
	}
	line.Handle(k(input.KeyLeft)) // back between x and h without forcing
	line.Handle(k(input.KeyRight))
	line.Handle(k(input.KeyHome)) // cursor before 'x'
	line.Handle(k(input.KeyDelete))
	if line.Text() != "hi!A" {
		t.Fatalf("delete at home: %q", line.Text())
	}
	line.Handle(k(input.KeyEnd))

	sc := drawAt(t, line, geom.Rect{Size: geom.Size{W: 40, H: 1}})
	if got := rowText(sc, 0, 0, 10); got != "Name: hi!A" {
		t.Fatalf("painted line %q", got)
	}
	// Block cursor: the cell right after the text (x=10: label occupies
	// 0..5, text starts at x=6, 4 runes) must carry the inverse block style.
	cur := sc.CellAt(10, 0)
	if cur.Rune != ' ' || cur.Style != styleInverse {
		t.Fatalf("cursor cell %+v is not an inverse block", cur)
	}

	if got := line.Handle(k(input.KeyEnter)); !got {
		t.Fatal("Enter must be consumed")
	}
	if ok != "hi!A" {
		t.Fatalf("onOK got %q, want %q", ok, "hi!A")
	}
}

func TestInputLineBackspaceRemovesChars(t *testing.T) {
	line := New("", "abc", nil, nil)
	for i := 0; i < 5; i++ {
		line.Handle(k(input.KeyBackspace))
	}
	if line.Text() != "" {
		t.Fatalf("backspace left %q", line.Text())
	}
	line.Handle(k(input.KeyBackspace)) // must not panic when empty
	line.SetValue("ready")
	if line.Text() != "ready" || line.CursorCol() != 5 {
		t.Fatalf("SetValue: text %q cursor %d", line.Text(), line.CursorCol())
	}

	// Nothing left on screen after every character is deleted.
	line.SetValue("zzz")
	line.Handle(k(input.KeyHome))
	for i := 0; i < 3; i++ {
		line.Handle(k(input.KeyDelete))
	}
	sc := drawAt(t, line, geom.Rect{Size: geom.Size{W: 20, H: 1}})
	if got := strings.TrimSpace(rowText(sc, 0, 0, 20)); got != "" {
		t.Fatalf("screen still shows %q after full delete", got)
	}
}

func TestInputLinePasteAndCancel(t *testing.T) {
	cancelled := 0
	line := New("> ", "", func(string) {}, func() { cancelled++ })
	line.Handle(input.Paste{Text: "hello"})
	line.Handle(k(input.KeyLeft))
	line.Handle(input.Paste{Text: "!"})
	if line.Text() != "hell!o" {
		t.Fatalf("paste: %q", line.Text())
	}
	if got := line.Handle(k(input.KeyEscape)); !got {
		t.Fatal("Esc must be consumed")
	}
	if cancelled != 1 {
		t.Fatalf("onCancel fired %d times", cancelled)
	}
	if line.Handle(input.Mouse{}) {
		t.Error("mouse events must not be consumed")
	}
	line.Blur()
	if line.Focused() {
		t.Error("Blur did not clear focused state")
	}
}

func TestInputLineNilCallbacksReturnFalse(t *testing.T) {
	line := New("", "", nil, nil)
	if line.Handle(k(input.KeyEnter)) || line.Handle(k(input.KeyEscape)) {
		t.Fatal("Enter/Esc with nil callbacks must not be consumed")
	}
}

func TestPopupCentering(t *testing.T) {
	child := &widget.Spacer{MinW: 10, MinH: 3}
	p := NewPopup("title", child)

	// Measure: 70% of a 60-wide parent with child height + 2 chrome rows.
	if got := p.Measure(geom.Size{W: 60}); got.W != 42 || got.H != 5 {
		t.Fatalf("popup measure = %v, want {42 5}", got)
	}
	// Minimum width floors at 20 cells.
	if got := p.Measure(geom.Size{W: 10}); got.W != 20 {
		t.Fatalf("popup minimum width = %d, want 20", got.W)
	}

	sc := drawAt(t, p, geom.Rect{Size: geom.Size{W: 40, H: 21}})
	// Frame is 28 wide (70% of 40) and 5 tall, centered inside 40x21:
	// columns 6..33, rows 8..12.
	left, right := sc.CellAt(6, 8), sc.CellAt(33, 8)
	if left.Rune != '╭' || right.Rune != '╮' {
		t.Fatalf("frame corners %q/%q are not centered rounded corners", left.Rune, right.Rune)
	}
	// Cells just outside the frame stay default blanks, so the frame is
	// really centered (not spanning the full width).
	if !sc.CellAt(5, 8).IsBlank() || !sc.CellAt(34, 8).IsBlank() {
		t.Error("popup frame is wider or off-center than expected")
	}
	// Side walls of the frame sit on the centered columns.
	if sc.CellAt(6, 10).Rune != '│' || sc.CellAt(33, 10).Rune != '│' {
		t.Error("frame side walls not at centered offsets")
	}
	// Cells far outside the frame stay blank.
	if !sc.CellAt(0, 0).IsBlank() || !sc.CellAt(10, 0).IsBlank() {
		t.Error("popup painted outside its centered frame")
	}
}

func TestPopupForwardsKeysNotMouse(t *testing.T) {
	rec := &recordingWidget{}
	p := NewPopup("x", rec)
	if !p.Handle(key('a')) || rec.count != 1 {
		t.Fatalf("keypress must reach the child, got consumed=%v count=%d", rec.count > 0, rec.count)
	}
	if p.Handle(input.Mouse{X: 1, Y: 1}) || rec.count != 1 {
		t.Fatal("mouse events must not be forwarded to the child")
	}
	if NewPopup("x", nil).Handle(key('a')) {
		t.Fatal("popup without child must not consume events")
	}
}

func TestWelcomeDrawSmallRect(t *testing.T) {
	w := &Welcome{Version: "0.1"}
	// Tiny rects must not panic and must stay inside the rect.
	for _, sz := range []geom.Size{{W: 0, H: 0}, {W: 2, H: 1}, {W: 80, H: 24}} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Welcome draw panicked on %v: %v", sz, r)
				}
			}()
			drawAt(t, w, geom.Rect{Size: sz})
		}()
	}

	sc := render.New(80, 24)
	w.Draw(&widget.DrawCtx{Rect: geom.Rect{Size: geom.Size{W: 80, H: 24}}, Screen: sc})
	found := false
	for y := 0; y < 24 && !found; y++ {
		if strings.Contains(rowText(sc, y, 0, 80), "P  A  N  D  A") {
			found = true
		}
	}
	if !found {
		t.Fatal("spaced editor name not painted")
	}
}

func TestDialogActions(t *testing.T) {
	var chosen []string
	d := NewDialog("Save changes?", []string{"yes", "no", "cancel"}, func(c string) { chosen = append(chosen, c) })

	// Enter fires the first action.
	if !d.Handle(k(input.KeyEnter)) || len(chosen) != 1 || chosen[0] != "yes" {
		t.Fatalf("Enter: consumed=%v chosen=%v", len(chosen) > 0, chosen)
	}
	// Esc dismisses with the empty string.
	d.Handle(k(input.KeyEscape))
	if len(chosen) != 2 || chosen[1] != "" {
		t.Fatalf("Esc: chosen=%v", chosen)
	}
	// Left then Right changes selection; Enter follows it.
	d.Handle(k(input.KeyLeft))
	d.Handle(k(input.KeyEnter))
	if chosen[len(chosen)-1] != "cancel" {
		t.Fatalf("left arrow should move to last action, got %q", chosen[len(chosen)-1])
	}
	d.Handle(k(input.KeyRight))
	d.Handle(k(input.KeyEnter))
	if chosen[len(chosen)-1] != "yes" {
		t.Fatalf("right arrow should wrap to first action, got %q", chosen[len(chosen)-1])
	}
	d.Handle(k(input.KeyRight))
	d.Handle(k(input.KeyEnter))
	if chosen[len(chosen)-1] != "no" {
		t.Fatalf("right arrow must select second action, got %q", chosen[len(chosen)-1])
	}
	// Letter presses pick the matching label.
	d.Handle(key('Y'))
	if chosen[len(chosen)-1] != "yes" {
		t.Fatalf("y key must pick yes, got %q", chosen[len(chosen)-1])
	}
	before := len(chosen)
	if d.Handle(key('z')) {
		t.Fatal("non-matching letter must be ignored")
	}
	if len(chosen) != before {
		t.Fatal("non-matching letter fired the callback")
	}
}

func TestDialogDrawHighlightsSelection(t *testing.T) {
	d := NewDialog("Really quit?", []string{"yes", "no"}, nil)
	sc := drawAt(t, d, geom.Rect{Size: geom.Size{W: 40, H: 6}})
	row := ""
	for y := 0; y < 6; y++ {
		row += rowText(sc, y, 0, 40)
	}
	if !strings.Contains(row, "Really quit?") {
		t.Error("message not painted")
	}
	if !strings.Contains(row, "yes") || !strings.Contains(row, "no") {
		t.Errorf("actions not painted: %q", row)
	}
	// Highlighted action: some cell must be reversed (accent bg over fg
	// swapped), i.e. carry the inverse style.
	reversed := false
	for y := 0; y < 6 && !reversed; y++ {
		for x := 0; x < 40; x++ {
			if sc.CellAt(x, y).Style == styleInverse {
				reversed = true
				break
			}
		}
	}
	if !reversed {
		t.Error("selected action is not rendered in inverse style")
	}
	// Background fill uses the palette's Indexed 236 surface throughout.
	if sc.CellAt(2, 1).Style.Bg != colBG || sc.CellAt(0, 0).Style.Bg != colBG {
		t.Error("dialog background is not the palette surface")
	}
}

func TestDialogNilCallbackDoesNotConsume(t *testing.T) {
	d := NewDialog("hi", []string{"yes", "no"}, nil)
	if d.Handle(k(input.KeyEnter)) || d.Handle(k(input.KeyEscape)) {
		t.Fatal("nil callback must leave Enter/Esc unconsumed")
	}
	// Drawing must not panic for degenerate rects either.
	drawAt(t, d, geom.Rect{Size: geom.Size{W: 1, H: 1}})
	drawAt(t, d, geom.Rect{})
}

// recordingWidget counts Handle calls for forwarding tests.
type recordingWidget struct {
	widget.Base
	count int
}

func (r *recordingWidget) Measure(geom.Size) geom.Size { return geom.Size{W: 10, H: 3} }
func (r *recordingWidget) Draw(*widget.DrawCtx)        {}
func (r *recordingWidget) Handle(input.Event) bool {
	r.count++
	return true
}
