package views

import (
	"fmt"
	"strings"

	"github.com/Aswanidev-vs/cherry/cell"
	"github.com/Aswanidev-vs/cherry/geom"
	"github.com/Aswanidev-vs/cherry/input"
	"github.com/Aswanidev-vs/cherry/widget"

	"github.com/Aswanidev-vs/panda-editor/editor/theme"
)

// StatusBarText builds the bottom bar segments: an instrument cluster with
// hairline dividers. Segments, left to right:
//   mode badge (uppercase, accent bold on raised, padded 1 cell each side)
//   │ file name
//   │ blinking ● modified dot in accent when dirty
//   │ [RO] badge in warning colour
//   │ transient message, centred, faint and italic, taking the flexible middle slot
//   │ right cluster: cursor position rendered compactly as 12:4 in bold with
//   │ faint LN/COL labels, then UTF-8 and LF.
//
// Returns segments ready for widget.NewStatusBar.
// ElidePath shortens a file path to fit budget columns, keeping the tail —
// the file name and its immediate parent are what identify a buffer, while
// the root of the path rarely is — and marking the cut with a leading
// ellipsis so a shortened path is never mistaken for the whole one.
//
// It is called with the space the status strip has left over after the mode
// badge and the readouts. Without it, a deeply nested path pushes the cursor
// position off the row, and the position is the one value in that strip a user
// glances at continuously.
func ElidePath(path string, budget int) string {
	if path == "" {
		return ""
	}
	if budget < 4 {
		budget = 4
	}
	if cell.StringWidth(path) <= budget {
		return path
	}
	runes := []rune(path)
	keep := budget - 1 // one cell for the ellipsis itself
	if keep > len(runes) {
		keep = len(runes)
	}
	// Prefer to start the tail at a path separator: a partial leading segment
	// like "c_editor\theme" is harder to read than a whole one.
	for i := len(runes) - keep; i < len(runes)-1; i++ {
		if runes[i] == '/' || runes[i] == '\\' {
			return string(theme.Glyph.Ellipsis) + string(runes[i+1:])
		}
	}
	return string(theme.Glyph.Ellipsis) + string(runes[len(runes)-keep:])
}

func StatusBarText(mode, file, message string, ln, col int, modified, readonly bool) []widget.Segment {
	// Mode badge: uppercase, on raised surface.
	modeBadge := strings.ToUpper(mode)
	if modeBadge == "" {
		modeBadge = " "
	}
	// File name.
	fileName := file
	if fileName == "" {
		fileName = " "
	}
	// Modified dot: blinking accent.
	var modDot cell.Cell
	r := theme.Current.Roles()
	if modified {
		modDot = cell.Cell{Rune: theme.Glyph.DotFilled, Style: r.Mark}
	} else {
		modDot = cell.Cell{Rune: ' '}
	}
	// Readonly badge.
	var roBadge string
	if readonly {
		roBadge = "[RO]"
	} else {
		roBadge = " "
	}
	// Right cluster: position, UTF-8, LF.
	// Position: compactly as 12:4.
	var pos string
	if ln > 0 && col > 0 {
		pos = fmt.Sprintf("%d:%d", ln, col)
	} else {
		pos = "  :"
	}
	// We'll make the entire right segment in ReadoutValue style (bold) for simplicity.
	// The design expects the position bold and the labels faint, but we cannot
	// achieve multi-styled text in a single segment. We'll make the position bold
	// and accept that the labels are also bold for now.
	rightCluster := fmt.Sprintf("%s  UTF-8  LF", pos)
	return []widget.Segment{
		{Text: " " + modeBadge + " ", Style: r.Badge},
		{Text: " " + fileName + " ", Style: r.Surface},
		{Text: string(modDot.Rune), Style: modDot.Style},
		{Text: " " + roBadge + " ", Style: r.ChromeLabel},
		{Text: message, Style: r.ChromeLabel, Flex: 1, Center: true},
		{Text: " " + rightCluster + " ", Style: r.ReadoutValue},
	}
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
	i.cursor = len(v)
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
	// Label in accent bold on base surface.
	roles := theme.Current.Roles()
	x := ctx.Screen.Print(r.Pos.X, r.Pos.Y, r.Right(), i.label, roles.Badge)
	// One separator space after the label, unless the label already ends
	// with one (or is empty).
	if i.label != "" && !strings.HasSuffix(i.label, " ") {
		x = ctx.Screen.Print(x, r.Pos.Y, r.Right(), " ", roles.Surface)
	}
	// Text in base style.
	ctx.Screen.Print(x, r.Pos.Y, r.Right(), string(i.runes), roles.Surface)

	// Block cursor: use the accent style for the cursor block.
	cx := x
	for _, ru := range i.runes[:i.cursor] {
		cx += cell.RuneWidth(ru)
	}
	cur := rune(' ')
	if i.cursor < len(i.runes) {
		cur = i.runes[i.cursor]
	}
	cursorBlock(ctx.Screen, cx, r.Pos.Y, r.Right(), cur, roles.Signal)
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
		// Delete the character before the cursor.
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

// Draw paints the popup centred inside ctx.Rect.
//
// It does not dim what is behind it. The scrim belongs to whoever owns the
// frame composition — the shell, which knows an overlay is coming and draws
// the editor first — because a container widget silently restyling every cell
// it was handed is a side effect no caller would expect, and it makes the
// widget untestable in isolation.
func (p *Popup) Draw(ctx *widget.DrawCtx) {
	r := ctx.Rect
	if r.Empty() {
		return
	}
	roles := theme.Current.Roles()
	// Popup dimensions.
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
	// Popup style: overlay surface background, dimmed accent frame.
	box := widget.Box{
		Mode:        widget.BorderRounded,
		Title:       p.Title,
		Background:  roles.Overlay,
		BorderStyle: theme.Style(theme.Current.AccentDim, roles.Overlay.Bg),
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

// Dialog is a small confirmation box with a centered message and two
// labelled actions; onAction receives the chosen label ("yes"/"no" etc.)
// or "" for Esc. Enter picks the first action, Esc cancels.
//
// Each action becomes a chip: [ label ], the selected chip in accent bold
// on the raised surface with a ▸ pointer before it, unselected chips in
// faint ink on the overlay surface. The dialog body is faint ink on the
// overlay surface.
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

// Measure reports the size the dialog needs: the wider of the message and the
// action row, plus the frame's border, its one-cell vertical padding, its
// two-cell horizontal padding, and the row the actions sit on.
//
// It must agree with Draw exactly. When the two disagree the dialog does not
// look wrong so much as lose content — the action row falls off the bottom —
// which is a genuinely confusing failure to debug from the screen.
func (d *Dialog) Measure(max geom.Size) geom.Size {
	msgW := 0
	for _, ln := range strings.Split(d.Message, "\n") {
		if lw := strW(ln); lw > msgW {
			msgW = lw
		}
	}
	actW := 0
	for i, a := range d.Actions {
		actW += strW(a) + 2 // the chip's brackets
		if i == d.sel {
			actW += cell.RuneWidth(theme.Glyph.PointerR) + 1
		}
	}
	if len(d.Actions) > 1 {
		actW += len(d.Actions) - 1 // the gaps between chips
	}
	w := msgW
	if actW > w {
		w = actW
	}
	h := len(strings.Split(d.Message, "\n")) + 1 // the actions' own row
	return fitSize(geom.Size{W: w + 6, H: h + 4}, max) // +2 borders, +2 padding
}

// Draw centres the dialog inside ctx.Rect at its measured size and paints it
// on the overlay surface.
//
// It centres rather than filling the assigned rect because the shell hands
// every overlay the whole viewport: a dialog that painted its rect as given
// would cover the entire editor and read as a crash, not a question.
func (d *Dialog) Draw(ctx *widget.DrawCtx) {
	r := ctx.Rect
	if r.Empty() {
		return
	}
	frame := d.frameIn(r)
	if frame.Empty() {
		return
	}

	roles := theme.Current.Roles()
	box := widget.Box{
		Mode:        widget.BorderRounded,
		Background:  roles.Overlay,
		BorderStyle: roles.Frame,
		PadT:        1,
		PadB:        1,
		PadL:        2,
		PadR:        2,
	}
	box.Draw(&widget.DrawCtx{Rect: frame, Screen: ctx.Screen})

	inner := geom.Rect{
		Pos:  geom.Point{X: frame.Pos.X + 3, Y: frame.Pos.Y + 2},
		Size: geom.Size{W: frame.Size.W - 5, H: frame.Size.H - 3},
	}
	if inner.Size.W <= 0 || inner.Size.H <= 0 {
		return
	}

	lines := widget.WrapText(d.Message, inner.Size.W)
	lastY := inner.Pos.Y
	for i, ln := range lines {
		y := inner.Pos.Y + i
		if y >= inner.Bottom() {
			lastY = inner.Bottom()
			break
		}
		printCentered(ctx.Screen, ln, y, inner, roles.DialogBody)
		lastY = y + 1
	}
	if len(d.Actions) == 0 {
		return
	}
	actionY := inner.Bottom() - 1
	if actionY < lastY {
		actionY = lastY
	}
	if actionY >= inner.Bottom() {
		return
	}

	total := 0
	for i, a := range d.Actions {
		total += strW(a) + 2 // the brackets
		if i == d.sel {
			total += cell.RuneWidth(theme.Glyph.PointerR) + 1
		}
	}
	if len(d.Actions) > 1 {
		total += len(d.Actions) - 1
	}
	x := inner.Pos.X + (inner.Size.W-total)/2
	if x < inner.Pos.X {
		x = inner.Pos.X
	}
	for i, a := range d.Actions {
		st := roles.ActionIdle
		if i == d.sel {
			st = roles.ActionFocus
			// The pointer sits in the chip's own style, so the whole
			// selection reads as one object rather than a glyph beside a
			// label.
			x = ctx.Screen.Print(x, actionY, inner.Right(), string(theme.Glyph.PointerR)+" ", st)
		}
		x = ctx.Screen.Print(x, actionY, inner.Right(), "["+a+"]", st)
		if i < len(d.Actions)-1 {
			x = ctx.Screen.Print(x, actionY, inner.Right(), " ", roles.Overlay)
		}
	}
}

// frameIn reports the centred rect the dialog occupies inside r: its measured
// size, clamped to r so a short terminal cannot push the frame off screen.
func (d *Dialog) frameIn(r geom.Rect) geom.Rect {
	pref := d.Measure(geom.Size{})
	w, h := pref.W, pref.H
	if w > r.Size.W {
		w = r.Size.W
	}
	if h > r.Size.H {
		h = r.Size.H
	}
	return geom.Rect{
		Pos:  geom.Point{X: r.Pos.X + (r.Size.W-w)/2, Y: r.Pos.Y + (r.Size.H-h)/2},
		Size: geom.Size{W: w, H: h},
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