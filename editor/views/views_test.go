package views

import (
	"strings"
	"testing"

	"github.com/Aswanidev-vs/cherry/cell"
	"github.com/Aswanidev-vs/cherry/geom"
	"github.com/Aswanidev-vs/cherry/input"
	"github.com/Aswanidev-vs/cherry/render"
	"github.com/Aswanidev-vs/cherry/widget"

	"github.com/Aswanidev-vs/panda-editor/editor/theme"
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
	if len(segs) != 6 {
		t.Fatalf("segments = %d, want 6", len(segs))
	}
	// segment 0: mode badge
	if !strings.Contains(segs[0].Text, "INSERT") {
		t.Errorf("mode missing from first segment %q", segs[0].Text)
	}
	if !strings.Contains(segs[0].Text, " ") {
		// mode badge should be padded with spaces
		t.Errorf("mode badge missing padding in %q", segs[0].Text)
	}
	// segment 1: file name
	if !strings.Contains(segs[1].Text, "main.go") {
		t.Errorf("file missing from second segment %q", segs[1].Text)
	}
	// segment 2: modified dot (should be space when false)
	if segs[2].Text != " " {
		t.Errorf("modified dot should be space when false, got %q", segs[2].Text)
	}
	// segment 3: readonly badge (should be space when false)
	if strings.TrimSpace(segs[3].Text) != "" {
		t.Errorf("readonly badge should be empty when false, got %q", segs[3].Text)
	}
	// segment 4: transient message
	if !strings.Contains(segs[4].Text, "saved") {
		t.Errorf("message missing from fifth segment %q", segs[4].Text)
	}
	// segment 5: right cluster (position, UTF-8, LF)
	if !strings.Contains(segs[5].Text, "3:12") {
		t.Errorf("position missing from sixth segment %q", segs[5].Text)
	}
	if !strings.Contains(segs[5].Text, "UTF-8") {
		t.Errorf("UTF-8 missing from sixth segment %q", segs[5].Text)
	}
	if !strings.Contains(segs[5].Text, "LF") {
		t.Errorf("LF missing from sixth segment %q", segs[5].Text)
	}

	mod := StatusBarText("edit", "notes.md", "", 1, 1, true, true)
	if len(mod) != 6 {
		t.Fatalf("modified segments = %d, want 6", len(mod))
	}
	// segment 0: mode badge
	if !strings.Contains(mod[0].Text, "EDIT") {
		t.Errorf("mode missing from first segment %q", mod[0].Text)
	}
	// segment 1: file name
	if !strings.Contains(mod[1].Text, "notes.md") {
		t.Errorf("file missing from second segment %q", mod[1].Text)
	}
	// segment 2: modified dot (should be dot when true)
	if mod[2].Text != string(theme.Glyph.DotFilled) {
		t.Errorf("modified dot missing in %q", mod[2].Text)
	}
	// segment 3: readonly badge (should be [RO] when true)
	if !strings.Contains(mod[3].Text, "[RO]") {
		t.Errorf("readonly marker missing in %q", mod[3].Text)
	}
	// segment 4: transient message (empty)
	if mod[4].Text != "" {
		t.Errorf("transient message should be empty when empty string, got %q", mod[4].Text)
	}
	// segment 5: right cluster (position, UTF-8, LF) - note: position is "1:1"
	if !strings.Contains(mod[5].Text, "1:1") {
		t.Errorf("position missing from sixth segment %q", mod[5].Text)
	}
	if !strings.Contains(mod[5].Text, "UTF-8") {
		t.Errorf("UTF-8 missing from sixth segment %q", mod[5].Text)
	}
	if !strings.Contains(mod[5].Text, "LF") {
		t.Errorf("LF missing from sixth segment %q", mod[5].Text)
	}

	// Empty inputs still yield six usable segments.
	empty := StatusBarText("", "", "", 0, 0, false, false)
	if len(empty) != 6 {
		t.Fatalf("empty segments = %d, want 6", len(empty))
	}
	// All segments should be empty strings or spaces etc.
	// We'll just check that they are not panicking.
}

func TestHintBar(t *testing.T) {
	for _, mode := range []string{"insert", "search", "dialog", "welcome"} {
		hb := HintBar(mode)
		if hb == nil {
			t.Fatalf("HintBar(%s) returned nil", mode)
		}
		// Check that the widget is a single row.
		if hb.Measure(geom.Size{}).H != 1 {
			t.Errorf("HintBar(%s) must be a single row", mode)
		}
		// We can also test the Draw by creating a screen and checking that it paints something.
		// We'll do a simple test: draw it and check that the background is Well (sunken).
		sc := render.New(20, 1)
		hb.Draw(&widget.DrawCtx{Rect: geom.Rect{Size: geom.Size{W: 20, H: 1}}, Screen: sc})
		// Check a few cells to see if they are sunken background.
		// We'll just check that the first cell is not the default background.
		// This is a weak test but better than nothing.
		if sc.CellAt(0, 0).Style.Bg == theme.Current.Roles().Surface.Bg {
			t.Error("HintBar background is not sunken")
		}
	}
	// Unknown mode should fall back to a generic strip.
	hb := HintBar("unknown")
	if hb == nil {
		t.Fatal("HintBar(\"unknown\") returned nil")
	}
	if hb.Measure(geom.Size{}).H != 1 {
		t.Error("HintBar(\"unknown\") must be a single row")
	}
	// We can also check that the content is not empty by drawing and seeing if it's not blank.
	sc := render.New(20, 1)
	hb.Draw(&widget.DrawCtx{Rect: geom.Rect{Size: geom.Size{W: 20, H: 1}}, Screen: sc})
	// Check that at least one cell is not blank.
	foundNonBlank := false
	for x := 0; x < 20; x++ {
		if !sc.CellAt(x, 0).IsBlank() {
			foundNonBlank = true
			break
		}
	}
	if !foundNonBlank {
		t.Error("HintBar(\"unknown\") appears to be blank")
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
	// 0..5, text starts at x=6, 4 runes) must carry the signal style.
	c := sc.CellAt(10, 0)
	roles := theme.Current.Roles()
	if c.Rune != ' ' || c.Style != roles.Signal {
		t.Fatalf("cursor cell %+v is not a signal block", c)
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

// TestDialogActions is unchanged from original.
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

// TestDialogDrawHighlightsSelection is updated for the new chip style.
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
	// Check the selected action: a chip carrying a ▸ pointer in the focus
	// style, with the label following the pointer and its gap.
	foundSelected := false
	r := theme.Current.Roles()
	for y := 0; y < 6 && !foundSelected; y++ {
		for x := 0; x < 40; x++ {
			c := sc.CellAt(x, y)
			if c.Rune != theme.Glyph.PointerR || c.Style != r.ActionFocus {
				continue
			}
			// The pointer is separated from the chip by one space, so the
			// '[' sits one cell further along than the pointer's own width.
			labelX := x + cell.RuneWidth(theme.Glyph.PointerR) + 1
			if labelX+1 < 40 &&
				sc.CellAt(labelX, y).Rune == '[' &&
				sc.CellAt(labelX+1, y).Rune == 'y' {
				foundSelected = true
				break
			}
		}
	}
	if !foundSelected {
		t.Error("selected action not found with pointer and focus style")
	}
	// Check that the unselected action is a chip in idle style without pointer.
	foundUnselected := false
	for y := 0; y < 6 && !foundUnselected; y++ {
		for x := 0; x < 40; x++ {
			c := sc.CellAt(x, y)
			if c.Rune == '[' && c.Style == r.ActionIdle {
				foundUnselected = true
				break
			}
		}
	}
	if !foundUnselected {
		t.Error("unselected action not found in idle style")
	}
	// The dialog is centred at its measured size, so its interior is a
	// known rect rather than the whole viewport. Assert the interior is the
	// overlay surface, and that the cells outside the frame are untouched —
	// the shell dims those itself, and a widget that painted over them would
	// hide the scrim that is supposed to make the dialog pop.
	top, left := -1, -1
	for y := 0; y < 6 && top < 0; y++ {
		for x := 0; x < 40; x++ {
			if sc.CellAt(x, y).Rune == '╭' {
				top, left = y, x
				break
			}
		}
	}
	if top < 0 {
		t.Fatal("dialog frame corner not found")
	}
	sz := d.Measure(geom.Size{})
	w, h := sz.W, sz.H
	interiorOK := true
	for y := top + 1; y < top+h-1 && interiorOK; y++ {
		for x := left + 1; x < left+w-1 && interiorOK; x++ {
			c := sc.CellAt(x, y)
			// The action chips deliberately sit on their own surfaces, so
			// only the body rows are expected to be flat overlay.
			if c.Style == r.ActionFocus || c.Style == r.ActionIdle {
				continue
			}
			if c.Style.Bg != r.Overlay.Bg {
				interiorOK = false
			}
		}
	}
	if !interiorOK {
		t.Error("dialog interior is not the overlay surface")
	}
	// Nothing may be painted outside the frame.
	outsideOK := true
	for y := 0; y < 6 && outsideOK; y++ {
		for x := 0; x < 40; x++ {
			inside := y >= top && y < top+h && x >= left && x < left+w
			if inside {
				continue
			}
			if c := sc.CellAt(x, y); c.Rune != ' ' && c.Rune != 0 {
				outsideOK = false
			}
		}
	}
	if !outsideOK {
		t.Error("dialog painted outside its centered frame")
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

// The path elision exists so the readouts are never the thing that gets
// dropped from the status strip, and so a shortened path is never mistaken
// for the whole one.
func TestElidePath(t *testing.T) {
	cases := []struct {
		name, in  string
		budget    int
		want      string
		suffix    string
		wantWidth int
	}{
		{name: "empty", in: "", budget: 20, want: ""},
		{name: "fits", in: `theme\style.go`, budget: 20, want: `theme\style.go`},
		{name: "exact fit", in: `theme\style.go`, budget: 15, want: `theme\style.go`},
		{
			name: "long windows path keeps the tail and marks the cut",
			in:   `G:\fastbeam\panda_editor\editor\theme\theme.go`, budget: 24,
			suffix: `theme\theme.go`,
		},
		{
			name: "a tiny budget still yields a marked path",
			in:   `G:\fastbeam\panda_editor\editor\theme\theme.go`, budget: 4,
			wantWidth: 4,
		},
		{
			name: "a nonsensical budget does not produce a negative width",
			in:   `some/very/long/path/indeed.go`, budget: 0,
			wantWidth: 4,
		},
		{
			name: "a unix path is cut the same way",
			in:   "/usr/local/src/editor/main.go", budget: 16,
			suffix:    "main.go",
			wantWidth: 16,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ElidePath(c.in, c.budget)
			if c.want != "" && got != c.want {
				t.Errorf("ElidePath(%q, %d) = %q, want %q", c.in, c.budget, got, c.want)
			}
			limit := c.wantWidth
			if limit == 0 {
				limit = c.budget
			}
			if w := cell.StringWidth(got); w > limit {
				t.Errorf("ElidePath(%q, %d) = %q (%d cells), want at most %d",
					c.in, c.budget, got, w, limit)
			}
			if c.suffix != "" && !strings.HasSuffix(got, c.suffix) {
				t.Errorf("ElidePath(%q, %d) = %q, want it to end in %q", c.in, c.budget, got, c.suffix)
			}
			if c.in != "" && got != c.in && []rune(got)[0] != theme.Glyph.Ellipsis {
				t.Errorf("ElidePath(%q, %d) = %q, want a leading ellipsis when it cuts", c.in, c.budget, got)
			}
		})
	}
}
