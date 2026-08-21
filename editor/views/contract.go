// Package views holds the editor's chrome widgets: the bottom status bar,
// the context hint bar, a single-line prompt input used by dialogs, a
// centered popup frame for confirmations, and the welcome splash. Each is a
// plain cherry widget with no knowledge of documents or files.
package views

import (
	"fmt"
	"strings"

	"github.com/Aswanidev-vs/cherry/cell"
	"github.com/Aswanidev-vs/cherry/geom"
	"github.com/Aswanidev-vs/cherry/input"
	"github.com/Aswanidev-vs/cherry/widget"
)

// StatusBarText builds the bottom bar segments: mode name + file name (+
// modified marker) on the left, message in the middle, "Ln X, Col Y" +
// encoding/EOL on the right. Returns segments ready for widget.NewStatusBar.
func StatusBarText(mode, file, message string, ln, col int, modified, readonly bool) []widget.Segment {
	left := strings.ToUpper(mode)
	if file != "" {
		if left != "" {
			left += " "
		}
		left += file
	}
	if modified {
		left += " [+]"
	}
	if readonly {
		left += " [RO]"
	}
	if left == "" {
		left = " "
	}
	right := fmt.Sprintf("Ln %d, Col %d  UTF-8  LF", ln, col)
	return []widget.Segment{
		{Text: " " + left + " ", Style: styleAccent},
		{Text: message, Style: styleDim, Flex: 1, Center: true},
		{Text: " " + right + " ", Style: styleDim},
	}
}

// HintBar renders the nano-style one-line key hint strip for the current
// mode. The mode picks which hints show (insert, search, dialog, welcome).
// Returning content is a fixed-height, left-aligned stripped strip that
// overflows from the right.
func HintBar(mode string) *widget.Text {
	hints, ok := hintStrips[mode]
	if !ok {
		hints = hintFallback
	}
	return &widget.Text{Content: hints, Style: styleDim}
}

// InputLine is a single-line edit prompt (goto-line, save-as, search). It
// shows a label, the raw text with a cursor, and handles typing, backspace,
// delete, home/end, arrows and paste. Enter/Esc are consumed and reported
// through the callbacks.
//
// Surface:
//
//	New(label string, initial string, onOK func(text string), onCancel func()) *InputLine
//	SetValue(v string)  — rearm for another query, cursor at end
//	Text() string
//	CursorCol() int     — rune column of cursor inside the text
//	Focus/Blur/Focused  — widget.Focusable
//
// Draw paints label + space + text on one line and marks the cursor as an
// inverse block cell. Handle consumes KeyPress: printable rune inserts at
// cursor; Backspace removes behind; Delete removes forward; Home/End/Left/
// Right move cursor; Enter calls onOK(text); Esc calls onCancel and returns
// true. Paste events insert the whole pasted text at cursor. Returns false
// for unrecognised events.
type InputLine struct {
	label    string
	runes    []rune
	cursor   int // rune offset inside runes; CursorCol reports it
	onOK     func(string)
	onCancel func()
	focused  bool
}

func New(label, initial string, onOK func(string), onCancel func()) *InputLine {
	runes := []rune(initial)
	return &InputLine{
		label:    label,
		runes:    runes,
		cursor:   len(runes),
		onOK:     onOK,
		onCancel: onCancel,
	}
}

func (i *InputLine) SetValue(v string) {
	i.runes = []rune(v)
	i.cursor = len(i.runes)
}

func (i *InputLine) Text() string   { return string(i.runes) }
func (i *InputLine) CursorCol() int { return i.cursor }

func (i *InputLine) Measure(max geom.Size) geom.Size {
	sep := 0
	if i.label != "" && !strings.HasSuffix(i.label, " ") {
		sep = 1
	}
	// +1 keeps room for the trailing block cursor.
	w := strW(i.label) + sep + strW(string(i.runes)) + 1
	return fitSize(geom.Size{W: w, H: 1}, max)
}

func (i *InputLine) Draw(ctx *widget.DrawCtx) {
	r := ctx.Rect
	if r.Empty() {
		return
	}
	ctx.Screen.Fill(r, blank(styleText))
	x := ctx.Screen.Print(r.Pos.X, r.Pos.Y, r.Right(), i.label, styleAccent)
	// One separator space after the label, unless the label already ends
	// with one (or is empty).
	if i.label != "" && !strings.HasSuffix(i.label, " ") {
		x = ctx.Screen.Print(x, r.Pos.Y, r.Right(), " ", styleText)
	}
	ctx.Screen.Print(x, r.Pos.Y, r.Right(), string(i.runes), styleText)

	// Block cursor: reverse the cell under the cursor (a space when the
	// cursor sits at the end) without disturbing the surrounding text.
	cx := x
	for _, ru := range i.runes[:i.cursor] {
		cx += cell.RuneWidth(ru)
	}
	cur := rune(' ')
	if i.cursor < len(i.runes) {
		cur = i.runes[i.cursor]
	}
	cursorBlock(ctx.Screen, cx, r.Pos.Y, r.Right(), cur, styleInverse)
}

func (i *InputLine) Handle(ev input.Event) bool {
	if p, ok := ev.(input.Paste); ok {
		i.insert(p.Text)
		return true
	}
	kp, ok := ev.(input.KeyPress)
	if !ok {
		return false
	}
	switch kp.Key {
	case input.KeyEnter:
		if i.onOK == nil {
			return false
		}
		i.onOK(string(i.runes))
		return true
	case input.KeyEscape:
		if i.onCancel == nil {
			return false
		}
		i.onCancel()
		return true
	case input.KeyBackspace:
		if i.cursor <= 0 {
			return true
		}
		i.runes = append(i.runes[:i.cursor-1], i.runes[i.cursor:]...)
		i.cursor--
		return true
	case input.KeyDelete:
		if i.cursor >= len(i.runes) {
			return true
		}
		i.runes = append(i.runes[:i.cursor], i.runes[i.cursor+1:]...)
		return true
	case input.KeyLeft:
		if i.cursor > 0 {
			i.cursor--
		}
		return true
	case input.KeyRight:
		if i.cursor < len(i.runes) {
			i.cursor++
		}
		return true
	case input.KeyHome:
		i.cursor = 0
		return true
	case input.KeyEnd:
		i.cursor = len(i.runes)
		return true
	case input.KeyNone:
		if kp.Rune == 0 || kp.Mod.Has(input.ModCtrl) || kp.Mod.Has(input.ModAlt) {
			return false
		}
		i.insert(string(kp.Rune))
		return true
	}
	return false
}

// insert splices s in at the cursor and moves the cursor past the input.
func (i *InputLine) insert(s string) {
	if s == "" {
		return
	}
	rs := []rune(s)
	tail := append([]rune(nil), i.runes[i.cursor:]...)
	i.runes = append(i.runes[:i.cursor], rs...)
	i.runes = append(i.runes, tail...)
	i.cursor += len(rs)
}

func (i *InputLine) Focus()        { i.focused = true }
func (i *InputLine) Blur()         { i.focused = false }
func (i *InputLine) Focused() bool { return i.focused }

// Popup is a centered frame carrying an arbitrary child with a title
// border. Its width is a fixed percentage of its parent rect (minimum 20
// cells), its height is the child's preferred height + chrome, vertically
// centered. Handle forwards only non-mouse events to the child.
type Popup struct {
	Title string
	Child widget.Widget
}

func NewPopup(title string, child widget.Widget) *Popup {
	return &Popup{Title: title, Child: child}
}

// popupWidth is 70% of the parent width with a 20-cell floor; unknown
// (unconstrained) parent widths fall back to the minimum.
func popupWidth(parentW int) int {
	if parentW <= 0 {
		return 20
	}
	if w := parentW * 70 / 100; w >= 20 {
		return w
	}
	return 20
}

func (p *Popup) Measure(max geom.Size) geom.Size {
	pw := popupWidth(max.W)
	var inner geom.Size
	if pw > 2 {
		inner.W = pw - 2
	}
	if max.H > 2 {
		inner.H = max.H - 2
	}
	var pref geom.Size
	if p.Child != nil {
		pref = p.Child.Measure(inner)
	}
	h := pref.H + 2
	if max.H > 0 && h > max.H {
		h = max.H
	}
	// Width intentionally ignores a smaller-than-minimum max: callers place
	// popups with Draw rects, which clamp the frame to the parent.
	return geom.Size{W: pw, H: h}
}

func (p *Popup) Draw(ctx *widget.DrawCtx) {
	r := ctx.Rect
	if r.Empty() {
		return
	}
	pw := popupWidth(r.Size.W)
	if pw > r.Size.W {
		pw = r.Size.W
	}
	var childH int
	if p.Child != nil {
		var inner geom.Size
		if pw > 2 {
			inner.W = pw - 2
		}
		if r.Size.H > 2 {
			inner.H = r.Size.H - 2
		}
		childH = p.Child.Measure(inner).H
	}
	ph := childH + 2
	if ph > r.Size.H {
		ph = r.Size.H
	}
	if pw < 2 {
		pw = r.Size.W
	}
	frame := geom.Rect{
		Pos:  geom.Point{X: r.Pos.X + (r.Size.W-pw)/2, Y: r.Pos.Y + (r.Size.H-ph)/2},
		Size: geom.Size{W: pw, H: ph},
	}
	box := widget.Box{
		Mode:        widget.BorderRounded,
		Title:       p.Title,
		Background:  styleText,
		BorderStyle: styleBorder,
		Child:       p.Child,
	}
	box.Draw(&widget.DrawCtx{Rect: frame, Screen: ctx.Screen})
}

func (p *Popup) Handle(ev input.Event) bool {
	if _, ok := ev.(input.Mouse); ok {
		return false
	}
	if p.Child == nil {
		return false
	}
	return p.Child.Handle(ev)
}

// Welcome is the splash widget shown when panda starts with no files: a
// spaced-distance large-format editor name, version and three principal
// shortcuts. Text only, no focus.
type Welcome struct {
	widget.Base
	Version string
}

func (w *Welcome) Measure(max geom.Size) geom.Size {
	rows := w.rows()
	wd := 0
	for _, row := range rows {
		if lw := strW(row.text); lw > wd {
			wd = lw
		}
	}
	return fitSize(geom.Size{W: wd, H: len(rows)}, max)
}

func (w *Welcome) Draw(ctx *widget.DrawCtx) {
	r := ctx.Rect
	if r.Empty() {
		return
	}
	rows := w.rows()
	y0 := r.Pos.Y + (r.Size.H-len(rows))/2
	if y0 < r.Pos.Y {
		y0 = r.Pos.Y
	}
	for k, row := range rows {
		y := y0 + k
		if y >= r.Bottom() {
			break
		}
		x := r.Pos.X + (r.Size.W-strW(row.text))/2
		if x < r.Pos.X {
			x = r.Pos.X
		}
		ctx.Screen.Print(x, y, r.Right(), row.text, row.style)
	}
}

// Dialog is a small confirmation box with a centered message and two
// labelled actions; onAction receives the chosen label ("yes"/"no" etc.)
// or "" for Esc. Enter picks the first action, Esc cancels.
type Dialog struct {
	Message  string
	Actions  []string
	OnAction func(choice string)

	sel int // currently highlighted action
}

func NewDialog(message string, actions []string, onAction func(string)) *Dialog {
	d := &Dialog{
		Message:  message,
		Actions:  append([]string(nil), actions...),
		OnAction: onAction,
	}
	d.clampSel()
	return d
}

func (d *Dialog) clampSel() {
	if len(d.Actions) == 0 || d.sel < 0 {
		d.sel = 0
		return
	}
	if d.sel >= len(d.Actions) {
		d.sel = len(d.Actions) - 1
	}
}

func (d *Dialog) Measure(max geom.Size) geom.Size {
	msgW := 0
	for _, ln := range strings.Split(d.Message, "\n") {
		if lw := strW(ln); lw > msgW {
			msgW = lw
		}
	}
	actW := 0
	for _, a := range d.Actions {
		actW += strW(a) + 2
	}
	if len(d.Actions) > 1 {
		actW += len(d.Actions) - 1
	}
	w := msgW
	if actW > w {
		w = actW
	}
	h := len(strings.Split(d.Message, "\n")) + 3
	return fitSize(geom.Size{W: w + 4, H: h}, max)
}

func (d *Dialog) Draw(ctx *widget.DrawCtx) {
	r := ctx.Rect
	if r.Empty() {
		return
	}
	frame := widget.Box{
		Mode:        widget.BorderRounded,
		Background:  styleText,
		BorderStyle: styleBorder,
	}
	frame.Draw(ctx)
	if r.Size.W <= 2 || r.Size.H <= 2 {
		return
	}
	inner := geom.Rect{
		Pos:  geom.Point{X: r.Pos.X + 1, Y: r.Pos.Y + 1},
		Size: geom.Size{W: r.Size.W - 2, H: r.Size.H - 2},
	}
	lines := widget.WrapText(d.Message, inner.Size.W)
	last := inner.Pos.Y
	for i, ln := range lines {
		y := inner.Pos.Y + i
		if y >= inner.Bottom() {
			last = inner.Bottom()
			break
		}
		printCentered(ctx.Screen, ln, y, inner, styleText)
		last = y + 1
	}
	if len(d.Actions) == 0 {
		return
	}
	y := inner.Bottom() - 1
	if y < last {
		y = last
	}
	if y >= inner.Bottom() {
		return
	}
	total := 0
	for _, a := range d.Actions {
		total += strW(a) + 2
	}
	if len(d.Actions) > 1 {
		total += len(d.Actions) - 1
	}
	x := inner.Pos.X + (inner.Size.W-total)/2
	if x < inner.Pos.X {
		x = inner.Pos.X
	}
	for i, a := range d.Actions {
		st := styleText
		if i == d.sel {
			st = styleInverse
		}
		x = ctx.Screen.Print(x, y, inner.Right(), " "+a+" ", st)
		if i < len(d.Actions)-1 {
			x = ctx.Screen.Print(x, y, inner.Right(), " ", styleText)
		}
	}
}

func (d *Dialog) Handle(ev input.Event) bool {
	kp, ok := ev.(input.KeyPress)
	if !ok {
		return false
	}
	switch kp.Key {
	case input.KeyEnter:
		if d.OnAction == nil {
			return false
		}
		choice := ""
		if len(d.Actions) > 0 {
			choice = d.Actions[d.sel]
		}
		d.OnAction(choice)
		return true
	case input.KeyEscape:
		if d.OnAction == nil {
			return false
		}
		d.OnAction("")
		return true
	case input.KeyLeft, input.KeyRight:
		if len(d.Actions) == 0 {
			return false
		}
		if kp.Key == input.KeyLeft {
			d.sel = (d.sel - 1 + len(d.Actions)) % len(d.Actions)
		} else {
			d.sel = (d.sel + 1) % len(d.Actions)
		}
		return true
	case input.KeyNone:
		if kp.Rune == 0 || d.OnAction == nil {
			return false
		}
		for i, a := range d.Actions {
			if actionMatchesKey(a, kp.Rune) {
				d.sel = i
				d.OnAction(a)
				return true
			}
		}
	}
	return false
}
