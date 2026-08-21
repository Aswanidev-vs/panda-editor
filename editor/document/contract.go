// Package document is the editor model: a textbuf.Buffer with a cursor,
// a selection, scroll position and read-only flag on top. It owns all
// editing semantics; UI widgets read its state and call its methods. It has
// no terminal or rendering concerns.
package document

import (
	"unicode"
	"unicode/utf8"

	"github.com/Aswanidev-vs/panda-editor/editor/textbuf"
)

// Pos aliases the buffer position so every layer shares one type.
type Pos = textbuf.Pos

// MoveDir selects a movement target for Move.
type MoveDir uint8

const (
	MoveLeft MoveDir = iota
	MoveRight
	MoveUp
	MoveDown
	MoveHome
	MoveEnd
	MoveWordLeft
	MoveWordRight
	MoveDocStart
	MoveDocEnd
)

// Document is one file being edited. All editing methods are single-line
// undo-grouped and refuse to mutate when ReadOnly is set.
//
// A Document is not safe for concurrent use; it is driven by one UI
// goroutine, the same rule as the textbuf.Buffer it wraps.
type Document struct {
	buf      *textbuf.Buffer
	path     string
	readOnly bool

	cursor     Pos
	anchor     Pos // selection endpoint fixed when the selection started
	hasSel     bool
	desiredCol int // remembered column goal for vertical movement

	scrollY int
}

// New returns an untitled empty document.
func New() *Document {
	return &Document{buf: textbuf.New()}
}

// Open loads path into a new document. A path that does not exist yields an
// empty document that remembers the path (saving creates it); a real read
// error is returned.
func Open(path string) (*Document, error) {
	b, err := textbuf.Open(path)
	if err != nil {
		// Missing file (or any read failure) starts an empty document
		// that remembers the path so Save can create it.
		return &Document{buf: textbuf.New(), path: path}, nil
	}
	return &Document{buf: b, path: path}, nil
}

// Path reports the file path; "" means untitled.
func (d *Document) Path() string { return d.path }

// SetPath repoints the document, e.g. after SaveAs.
func (d *Document) SetPath(path string) { d.path = path }

// Buffer exposes the underlying text buffer for position arithmetic, search
// and any operation the model does not wrap directly.
func (d *Document) Buffer() *textbuf.Buffer { return d.buf }

// ReadOnly reports / SetReadOnly controls edit protection. Movement still
// works when read-only; editing methods become no-ops.
func (d *Document) ReadOnly() bool     { return d.readOnly }
func (d *Document) SetReadOnly(v bool) { d.readOnly = v }
func (d *Document) Modified() bool     { return d.buf.Modified() }

// clampPos rounds p through the buffer so the column never points inside a
// rune or past the end of its line.
func (d *Document) clampPos(p Pos) Pos {
	return d.buf.OffsetToPos(d.buf.PosToOffset(p))
}

// edge returns the end-of-document position (last line, its end).
func (d *Document) edge() Pos {
	line := d.buf.LineCount() - 1
	return Pos{Line: line, Col: d.buf.LineLen(line)}
}

// Cursor returns the cursor position; SetCursor moves it (clamped). A bare
// SetCursor clears any selection.
func (d *Document) Cursor() Pos { return d.cursor }

// SetCursor clamps p, moves the cursor there, clears the selection and
// resets the column memory.
func (d *Document) SetCursor(p Pos) {
	d.cursor = d.clampPos(p)
	d.hasSel = false
	d.desiredCol = d.cursor.Col
}

// SelectTo moves the cursor to p while keeping (or starting) the selection
// anchor at the previous cursor position.
func (d *Document) SelectTo(p Pos) {
	if !d.hasSel {
		d.anchor = d.cursor
	}
	d.hasSel = true
	d.cursor = d.clampPos(p)
	// desiredCol is intentionally untouched so shift+vertical runs keep
	// the column goal.
}

// SelectNone drops the selection, keeping the cursor where it is.
func (d *Document) SelectNone() {
	d.hasSel = false
	d.desiredCol = d.cursor.Col
}

// SelectAll anchors document start and moves the cursor to document end.
func (d *Document) SelectAll() {
	d.anchor = Pos{}
	d.cursor = d.clampPos(d.edge())
	d.hasSel = true
	d.desiredCol = d.cursor.Col
}

// Selection returns the normalized ordered endpoints of the current
// selection; ok is false when there is none.
func (d *Document) Selection() (start, end Pos, ok bool) {
	if !d.hasSel {
		return Pos{}, Pos{}, false
	}
	a, b := d.anchor, d.cursor
	if d.buf.PosToOffset(b) < d.buf.PosToOffset(a) {
		a, b = b, a
	}
	return a, b, true
}

// SelectionText returns the selected text, "" when nothing is selected.
func (d *Document) SelectionText() string {
	start, end, ok := d.Selection()
	if !ok {
		return ""
	}
	s := d.buf.PosToOffset(start)
	e := d.buf.PosToOffset(end)
	return d.buf.Text()[s:e]
}

// SelByteOffsets returns the selection as byte offsets, ok false otherwise.
func (d *Document) SelByteOffsets() (start, end int, ok bool) {
	s, e, ok := d.Selection()
	if !ok {
		return 0, 0, false
	}
	return d.buf.PosToOffset(s), d.buf.PosToOffset(e), true
}

// deleteSelection removes the selected range (when it covers any text) and
// leaves the cursor at the selection start. It reports whether a selection
// existed at all.
func (d *Document) deleteSelection() bool {
	if !d.hasSel {
		return false
	}
	s, e, _ := d.Selection()
	o1 := d.buf.PosToOffset(s)
	o2 := d.buf.PosToOffset(e)
	if o2 > o1 {
		d.buf.DeleteRange(o1, o2)
	}
	d.cursor = d.buf.OffsetToPos(o1)
	d.hasSel = false
	d.desiredCol = d.cursor.Col
	return true
}

// began returns false (and does nothing else) when the document is
// read-only; otherwise it opens one undo group and returns true.
func (d *Document) began() bool {
	if d.readOnly {
		return false
	}
	d.buf.BeginGroup()
	return true
}

// InsertRune / InsertText type at the cursor, first deleting any selection.
// InsertNewline breaks the line and repeats the current line's leading
// whitespace on the new one. Editing methods are one undo step each.
func (d *Document) InsertRune(r rune) {
	if d.readOnly {
		return
	}
	d.deleteSelection()
	off := d.buf.PosToOffset(d.cursor)
	d.buf.InsertRune(d.cursor, r)
	d.setOffsetCursor(off + len(string(r)))
}

// InsertText inserts the whole string at the cursor, cursor past the
// inserted content.
func (d *Document) InsertText(s string) {
	if d.readOnly {
		return
	}
	d.deleteSelection()
	if s == "" {
		return
	}
	off := d.buf.PosToOffset(d.cursor)
	d.buf.InsertPos(d.cursor, s)
	d.setOffsetCursor(off + len(s))
}

// InsertNewline expands '\n', auto-indenting by repeating the current
// line's leading whitespace.
func (d *Document) InsertNewline() {
	if d.readOnly {
		return
	}
	d.deleteSelection()
	line := d.buf.Line(d.cursor.Line)
	indent := leadingWhitespace(line)
	off := d.buf.PosToOffset(d.cursor)
	d.buf.InsertPos(d.cursor, "\n"+indent)
	d.setOffsetCursor(off + 1 + len(indent))
}

// setOffsetCursor puts the cursor at byte offset off (clamped and snapped),
// clears the selection and resyncs the column memory.
func (d *Document) setOffsetCursor(off int) {
	d.cursor = d.buf.OffsetToPos(off)
	d.hasSel = false
	d.desiredCol = d.cursor.Col
}

func leadingWhitespace(s string) string {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return s[:i]
}

// Indent / Dedent operate on every line the selection touches (one tab or
// four-space equivalent per line); with no selection Indent inserts a tab
// at the cursor.
func (d *Document) Indent() {
	if d.readOnly {
		return
	}
	if d.hasSel {
		s, e, _ := d.Selection()
		if s.Line != e.Line {
			last := e.Line
			if e.Col == 0 {
				last-- // a caret on column 0 does not touch this line
			}
			if !d.began() {
				return
			}
			for ln := s.Line; ln <= last; ln++ {
				d.buf.InsertPos(Pos{Line: ln}, "\t")
			}
			d.buf.EndGroup()
			d.resettle()
			return
		}
	}
	// No selection or a single-line selection: replace the selection (if
	// any) with one tab at its start.
	d.deleteSelection()
	off := d.buf.PosToOffset(d.cursor)
	d.buf.InsertPos(d.cursor, "\t")
	d.setOffsetCursor(off + 1)
}

// Dedent strips one leading tab - or up to four leading spaces when the
// line starts with fewer - from every line the selection touches, or from
// the cursor's line when nothing is selected.
func (d *Document) Dedent() {
	if !d.began() {
		return
	}
	s, e, _ := d.Selection()
	if !d.hasSel {
		s, e = d.cursor, d.cursor
	} else if s.Line == e.Line {
		e = s // single-line selection: only its line is touched
	} else if e.Col == 0 {
		e = Pos{Line: e.Line - 1} // caret on column 0 does not touch the line
	}
	for ln := s.Line; ln <= e.Line; ln++ {
		d.dedentLine(ln)
	}
	d.buf.EndGroup()
	d.resettle()
}

// dedentLine removes the leading tab or up to four leading spaces of line.
func (d *Document) dedentLine(line int) {
	lb := d.buf.Line(line)
	start := d.buf.PosToOffset(Pos{Line: line})
	if len(lb) > 0 && lb[0] == '\t' {
		d.buf.DeleteRange(start, start+1)
		return
	}
	n := 0
	for n < 4 && n < len(lb) && lb[n] == ' ' {
		n++
	}
	if n > 0 {
		d.buf.DeleteRange(start, start+n)
	}
}

// resettle re-clamps cursor and anchor after a bulk line edit and resyncs
// the column memory.
func (d *Document) resettle() {
	d.cursor = d.clampPos(d.cursor)
	d.anchor = d.clampPos(d.anchor)
	d.desiredCol = d.cursor.Col
}

// Backspace deletes the selection, or one rune backward, or joins lines at
// column zero. DeleteForward mirrors it ahead of the cursor.
func (d *Document) Backspace() {
	if d.readOnly {
		return
	}
	if d.hasSel {
		d.deleteSelection()
		return
	}
	end := d.buf.PosToOffset(d.cursor)
	if end == 0 {
		return
	}
	start := end - 1
	for start > 0 && d.buf.Text()[start]&0xC0 == 0x80 { // snap onto the rune start
		start--
	}
	d.buf.DeleteRange(start, end)
	d.setOffsetCursor(start)
}

// DeleteForward deletes the selection, or one rune ahead of the cursor; at
// a line end it joins the next line.
func (d *Document) DeleteForward() {
	if d.readOnly {
		return
	}
	if d.hasSel {
		d.deleteSelection()
		return
	}
	d.buf.DeletePos(d.cursor, 1) // '\n' counts as one rune, so this joins lines too
}

// DeleteWordBackward / Forward remove the word adjacent to the cursor.
// Word scans stop at line edges: at column 0 (or the line end) they no-op
// instead of joining lines.
func (d *Document) DeleteWordBackward() {
	if d.readOnly {
		return
	}
	if d.hasSel {
		d.deleteSelection()
		return
	}
	end := d.buf.PosToOffset(d.cursor)
	lb := d.buf.Line(d.cursor.Line)
	pre := []rune(lb[:byteIndexForCol(lb, d.cursor.Col)])
	i := len(pre)
	for i > 0 && !isWordRune(pre[i-1]) {
		i--
	}
	for i > 0 && isWordRune(pre[i-1]) {
		i--
	}
	start := end - len(string(pre[i:]))
	if start == end {
		return
	}
	d.buf.DeleteRange(start, end)
	d.setOffsetCursor(start)
}

// DeleteWordForward removes the word ahead of the cursor within its line.
func (d *Document) DeleteWordForward() {
	if d.readOnly {
		return
	}
	if d.hasSel {
		d.deleteSelection()
		return
	}
	start := d.buf.PosToOffset(d.cursor)
	lb := d.buf.Line(d.cursor.Line)
	suf := []rune(lb[byteIndexForCol(lb, d.cursor.Col):])
	i := 0
	for i < len(suf) && !isWordRune(suf[i]) {
		i++
	}
	for i < len(suf) && isWordRune(suf[i]) {
		i++
	}
	end := start + len(string(suf[:i]))
	if end == start {
		return
	}
	d.buf.DeleteRange(start, end)
}

// byteIndexForCol returns the byte index inside line s of rune column col
// (clamped to the end of the line).
func byteIndexForCol(s string, col int) int {
	i := 0
	for c := 0; c < col && i < len(s); c++ {
		_, sz := utf8.DecodeRuneInString(s[i:])
		i += sz
	}
	return i
}

// isWordRune reports whether r belongs to a word run: a unicode letter or
// digit.
func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

// Undo / Redo drive the buffer history.
func (d *Document) Undo() bool {
	if d.readOnly {
		return false
	}
	return d.applyHistory(d.buf.Undo())
}

// Redo reapplies the most recently undone step.
func (d *Document) Redo() bool {
	if d.readOnly {
		return false
	}
	return d.applyHistory(d.buf.Redo())
}

// applyHistory reloads cursor and anchor through a round trip after an
// undo/redo so both stay inside the resized buffer.
func (d *Document) applyHistory(ok bool) bool {
	if !ok {
		return false
	}
	d.cursor = d.clampPos(d.cursor)
	d.anchor = d.clampPos(d.anchor)
	d.desiredCol = d.cursor.Col
	return true
}

// ReplaceRange replaces [start,end) bytes with s as one undo step and moves
// the cursor to just past the replacement; it returns that offset. It is the
// primitive search operations build on.
func (d *Document) ReplaceRange(start, end int, s string) int {
	n := d.buf.Len()
	if start > end {
		start, end = end, start
	}
	if start < 0 {
		start = 0
	} else if start > n {
		start = n
	}
	if end < 0 {
		end = 0
	} else if end > n {
		end = n
	}
	d.buf.BeginGroup()
	d.buf.DeleteRange(start, end)
	d.buf.Insert(start, s)
	d.buf.EndGroup()
	d.SetCursor(d.buf.OffsetToPos(start + len(s)))
	return start + len(s)
}

// Move moves the cursor in dir, optionally extending the selection instead
// of clearing it. Column memory survives vertical moves across short lines.
func (d *Document) Move(dir MoveDir, sel bool) {
	cur := d.cursor
	var target Pos
	vertical := false
	switch dir {
	case MoveLeft:
		if cur.Col == 0 && cur.Line > 0 {
			prev := cur.Line - 1
			target = Pos{Line: prev, Col: d.buf.LineLen(prev)}
		} else {
			target = Pos{Line: cur.Line, Col: cur.Col - 1}
		}
	case MoveRight:
		if cur.Col == d.buf.LineLen(cur.Line) && cur.Line < d.buf.LineCount()-1 {
			target = Pos{Line: cur.Line + 1}
		} else {
			target = Pos{Line: cur.Line, Col: cur.Col + 1}
		}
	case MoveUp:
		target = Pos{Line: cur.Line - 1, Col: d.desiredCol}
		vertical = true
	case MoveDown:
		target = Pos{Line: cur.Line + 1, Col: d.desiredCol}
		vertical = true
	case MoveHome:
		target = Pos{Line: cur.Line}
	case MoveEnd:
		target = Pos{Line: cur.Line, Col: d.buf.LineLen(cur.Line)}
	case MoveWordLeft:
		target = d.buf.OffsetToPos(d.wordTarget(false))
	case MoveWordRight:
		target = d.buf.OffsetToPos(d.wordTarget(true))
	case MoveDocStart:
		target = Pos{}
	case MoveDocEnd:
		target = d.edge()
	default:
		return
	}
	d.place(target, vertical, sel)
}

// place moves the cursor to p, via SelectTo when sel, and resets the column
// memory unless the move was vertical (which keeps it).
func (d *Document) place(p Pos, keepCol, sel bool) {
	col := d.desiredCol
	if sel {
		d.SelectTo(p)
	} else {
		d.SetCursor(p)
	}
	if keepCol {
		d.desiredCol = col
	}
}

// wordTarget computes the MoveWordLeft / MoveWordRight landing offset by
// scanning unicode letter/digit runs through the whole document.
func (d *Document) wordTarget(forward bool) int {
	s := d.buf.Text()
	off := d.buf.PosToOffset(d.cursor)
	var offs []int
	rs := make([]rune, 0, len(s))
	for i, r := range s {
		offs = append(offs, i)
		rs = append(rs, r)
	}
	offs = append(offs, len(s))
	k := 0
	for k < len(offs) && offs[k] < off {
		k++
	}
	if k < len(offs) && offs[k] > off && k > 0 {
		k-- // off landed mid-rune; fall back onto the rune start
	}
	i := k
	if forward {
		for i < len(rs) && !isWordRune(rs[i]) {
			i++
		}
		for i < len(rs) && isWordRune(rs[i]) {
			i++
		}
	} else {
		for i > 0 && !isWordRune(rs[i-1]) {
			i--
		}
		for i > 0 && isWordRune(rs[i-1]) {
			i--
		}
	}
	return offs[i]
}

// MoveLines jumps n lines vertically (negative up, positive down), keeping
// the column memory; paging uses it with the viewport height.
func (d *Document) MoveLines(n int, sel bool) {
	d.place(Pos{Line: d.cursor.Line + n, Col: d.desiredCol}, true, sel)
}

// GotoLine moves to line ln (0-based), column 0.
func (d *Document) GotoLine(ln int) {
	d.SetCursor(Pos{Line: ln})
}

// ScrollY / SetScrollY manage the top visible line, clamped to the buffer.
// EnsureCursorVisible scrolls the minimum needed so the cursor row sits
// inside a viewport of viewH rows.
func (d *Document) ScrollY() int { return d.scrollY }

// SetScrollY clamps y into 0..LineCount-1.
func (d *Document) SetScrollY(y int) {
	d.scrollY = d.clampScroll(y)
}

// EnsureCursorVisible scrolls the minimum so the cursor line is inside a
// viewport of viewH rows.
func (d *Document) EnsureCursorVisible(viewH int) {
	line := d.cursor.Line
	if line < d.scrollY {
		d.scrollY = line
	} else if viewH > 0 && line >= d.scrollY+viewH {
		d.scrollY = line - viewH + 1
	}
	d.scrollY = d.clampScroll(d.scrollY)
}

func (d *Document) clampScroll(y int) int {
	if y < 0 {
		return 0
	}
	if max := d.buf.LineCount() - 1; y > max {
		return max
	}
	return y
}

// LineCount forwards the buffer's line count for painting loops.
func (d *Document) LineCount() int { return d.buf.LineCount() }

// Text forwards the buffer's contents for callers that want the whole
// document as one string.
func (d *Document) Text() string { return d.buf.Text() }
