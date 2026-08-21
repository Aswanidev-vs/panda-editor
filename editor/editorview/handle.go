package editorview

import (
	"github.com/Aswanidev-vs/cherry/geom"
	"github.com/Aswanidev-vs/cherry/input"

	"github.com/Aswanidev-vs/panda-editor/editor/document"
)

func (v *View) Handle(ev input.Event) bool {
	if v.doc == nil {
		return false
	}
	switch e := ev.(type) {
	case input.KeyPress:
		return v.handleKey(e)
	case input.Mouse:
		return v.handleMouse(e)
	case input.Paste:
		if e.Text == "" {
			return false
		}
		if !v.doc.ReadOnly() {
			v.doc.SelectNone()
			v.doc.InsertText(e.Text)
			v.updateScroll()
		}
		return true
	case input.Resize:
		v.doc.EnsureCursorVisible(e.Height)
		return false
	}
	return false
}

func (v *View) handleKey(e input.KeyPress) bool {
	doc := v.doc
	if v.modal != nil && v.modal.HandleKey(e, doc) {
		return true
	}
	shift := e.Mod.Has(input.ModShift)
	ctrl := e.Mod.Has(input.ModCtrl)
	ro := doc.ReadOnly()

	if e.Key == input.KeyNone {
		switch {
		case ctrl:
			r := e.Rune
			if r >= 1 && r <= 26 {
				r += 'a' - 1
			}
			return v.keyCtrlRune(r, shift)
		case e.Rune != 0:
			if !ro {
				doc.SelectNone()
				doc.InsertRune(e.Rune)
				v.updateScroll()
			}
			return true
		}
		return false
	}

	switch e.Key {
	case input.KeyEnter:
		if ro {
			doc.Move(document.MoveDown, false)
		} else {
			doc.SelectNone()
			doc.InsertNewline()
		}
		v.updateScroll()
		return true
	case input.KeyTab, input.KeyBacktab:
		if !ro {
			if e.Key == input.KeyBacktab || shift {
				doc.Dedent()
			} else {
				doc.Indent()
			}
			v.updateScroll()
		}
		return true
	case input.KeyBackspace:
		if !ro {
			if ctrl {
				doc.DeleteWordBackward()
			} else {
				doc.Backspace()
			}
			v.updateScroll()
		}
		return true
	case input.KeyDelete:
		if !ro {
			if ctrl {
				doc.DeleteWordForward()
			} else {
				doc.DeleteForward()
			}
			v.updateScroll()
		}
		return true
	case input.KeyPageUp:
		doc.MoveLines(-v.viewRows(), shift)
		v.updateScroll()
		return true
	case input.KeyPageDown:
		doc.MoveLines(v.viewRows(), shift)
		v.updateScroll()
		return true
	case input.KeyEscape:
		return false
	}

	if dir, ok := moveDir(e.Key, ctrl); ok {
		doc.Move(dir, shift)
		v.updateScroll()
		return true
	}
	return false
}

func moveDir(k input.Key, ctrl bool) (document.MoveDir, bool) {
	switch k {
	case input.KeyLeft:
		if ctrl {
			return document.MoveWordLeft, true
		}
		return document.MoveLeft, true
	case input.KeyRight:
		if ctrl {
			return document.MoveWordRight, true
		}
		return document.MoveRight, true
	case input.KeyUp:
		return document.MoveUp, true
	case input.KeyDown:
		return document.MoveDown, true
	case input.KeyHome:
		if ctrl {
			return document.MoveDocStart, true
		}
		return document.MoveHome, true
	case input.KeyEnd:
		if ctrl {
			return document.MoveDocEnd, true
		}
		return document.MoveEnd, true
	}
	return 0, false
}

func (v *View) keyCtrlRune(r rune, shift bool) bool {
	doc := v.doc
	switch r {
	case 'z', 'Z':
		if shift {
			doc.Redo()
		} else {
			doc.Undo()
		}
		v.updateScroll()
		return true
	case 'y', 'Y':
		doc.Redo()
		v.updateScroll()
		return true
	case 'a', 'A':
		doc.SelectAll()
		v.updateScroll()
		return true
	case 'x', 'X':
		v.cutSelection()
		return true
	case 'c', 'C':
		v.copySelection()
		return true
	case 'v', 'V':
		v.pasteClipboard()
		return true
	case 'w', 'W':
		if !doc.ReadOnly() {
			doc.DeleteWordBackward()
			v.updateScroll()
		}
		return true
	}
	return false
}

func (v *View) cutSelection() {
	start, end, ok := v.doc.SelByteOffsets()
	if !ok {
		return
	}
	if text := v.doc.SelectionText(); text != "" && v.app != nil {
		v.app.SetClipboard(text)
	}
	if !v.doc.ReadOnly() {
		v.doc.ReplaceRange(start, end, "")
		v.updateScroll()
	}
}

func (v *View) copySelection() {
	if v.app == nil {
		return
	}
	if text := v.doc.SelectionText(); text != "" {
		v.app.SetClipboard(text)
	}
}

func (v *View) pasteClipboard() {
	if v.app == nil || v.doc.ReadOnly() {
		return
	}
	text := v.app.ClipboardText()
	if text == "" {
		return
	}
	v.doc.SelectNone()
	v.doc.InsertText(text)
	v.updateScroll()
}

func (v *View) handleMouse(e input.Mouse) bool {
	if !v.drawn {
		return false
	}
	rect := v.lastRect
	if !rect.Contains(geom.Point{X: e.X, Y: e.Y}) {
		return false
	}
	doc := v.doc
	switch e.Action {
	case input.MousePress:
		switch e.Button {
		case 0:
			p := v.posAt(e.X, e.Y)
			if e.Mod.Has(input.ModShift) {
				doc.SelectTo(p)
			} else {
				doc.SetCursor(p)
			}
			v.dragging = true
			v.updateScroll()
			return true
		case 1:
			if v.app != nil {
				if text := v.app.ClipboardText(); text != "" && !doc.ReadOnly() {
					doc.SelectNone()
					doc.InsertText(text)
					v.updateScroll()
				}
			}
			return true
		default:
			return true
		}
	case input.MouseMove:
		if !v.dragging || e.Button != 0 {
			return false
		}
		doc.SelectTo(v.posAt(e.X, e.Y))
		v.updateScroll()
		return true
	case input.MouseRelease:
		if v.dragging {
			v.dragging = false
			return true
		}
		return false
	case input.MouseWheel:
		dy := 3
		if e.Wheel == input.WheelUp {
			dy = -3
		}
		doc.SetScrollY(doc.ScrollY() + dy)
		doc.EnsureCursorVisible(v.viewRows())
		return true
	}
	return false
}

// posAt maps a screen cell to the nearest document position.
func (v *View) posAt(mx, my int) document.Pos {
	doc := v.doc
	rect := v.lastRect
	line := doc.ScrollY() + (my - rect.Pos.Y)
	lc := doc.LineCount()
	if line < 0 {
		line = 0
	}
	if line >= lc {
		line = lc - 1
	}
	gw := digitCount(lc) + 1
	textX := rect.Pos.X + gw + 1
	col := 0
	if mx >= textX {
		row := v.hitRow(line)
		col = hitCol(buildSegs(row, textX, rect.Right(), v.tabWidth), mx)
		if col > row.nrunes {
			col = row.nrunes
		}
	}
	return document.Pos{Line: line, Col: col}
}

// hitRow returns the lexed row for hit testing, building it on demand.
func (v *View) hitRow(lineNo int) *lexRow {
	doc := v.doc
	lc := doc.LineCount()
	v.growRows(lc)
	states := v.chainTo(doc, lineNo+1)
	row := &v.rows[lineNo]
	line := doc.Buffer().Line(lineNo)
	if !row.lexed || row.line != line || row.in != states[lineNo] {
		v.lexRow(line, states[lineNo], row)
	}
	return row
}

func hitCol(segs []seg, x int) int {
	for _, s := range segs {
		if x < s.x {
			return s.ri
		}
		if x < s.x+s.w {
			if 2*(x-s.x) < s.w {
				return s.ri
			}
			return s.ri + 1
		}
	}
	if len(segs) == 0 {
		return 0
	}
	return segs[len(segs)-1].ri + 1
}

func (v *View) viewRows() int {
	if v.drawn {
		return v.lastRect.Size.H
	}
	return 24
}

func (v *View) updateScroll() {
	if v.doc != nil {
		v.doc.EnsureCursorVisible(v.viewRows())
	}
}

// CursorPos reports the cursor cell from the last drawn rect; hidden when
// the view is unfocused or the cursor line scrolled out of the viewport.
func (v *View) CursorPos() (geom.Point, bool) {
	if !v.focused || v.doc == nil || !v.drawn || v.lastRect.Empty() {
		return geom.Point{}, false
	}
	doc := v.doc
	cur := doc.Cursor()
	dy := cur.Line - doc.ScrollY()
	if dy < 0 || dy >= v.lastRect.Size.H {
		return geom.Point{}, false
	}
	gw := digitCount(doc.LineCount()) + 1
	textX := v.lastRect.Pos.X + gw + 1
	line := doc.Buffer().Line(cur.Line)
	x := runWidthTo(line, cur.Col, textX, v.tabWidth)
	return geom.Point{X: x, Y: v.lastRect.Pos.Y + dy}, true
}
