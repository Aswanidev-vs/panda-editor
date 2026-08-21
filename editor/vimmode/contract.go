// Package vimmode is the opt-in modal editing layer. Installed on the
// editorview via SetModal, it intercepts keys before insert-mode handling
// and implements normal / insert / visual / visual-line / replace states
// over the document's editing verbs. The shell toggles it with vim_mode.
package vimmode

import (
	"strconv"
	"strings"

	"github.com/Aswanidev-vs/cherry/input"

	"github.com/Aswanidev-vs/panda-editor/editor/document"
)

// State enumerates modal states for status display.
type State uint8

const (
	Normal State = iota
	Insert
	Visual
	VisualLine
	Replace
)

// String reports the state name used by chrome ("NORMAL", "INSERT", ...).
func (s State) String() string {
	switch s {
	case Normal:
		return "NORMAL"
	case Insert:
		return "INSERT"
	case Visual:
		return "VISUAL"
	case VisualLine:
		return "V-LINE"
	case Replace:
		return "REPLACE"
	}
	return "?"
}

type operator uint8

const (
	opNone operator = 0
	opDelete operator = 'd'
	opChange operator = 'c'
	opYank   operator = 'y'
)

// Mode is the layer the shell installs via editorview.SetModal. A zero value
// is not usable: construct with New.
type Mode struct {
	state      State
	count      int
	op         operator
	replaceR   rune
	haveReplace bool
	register   string
	pendingGoto int
	lastYank   string
}

// New returns a fresh modal layer starting in Normal state.
func New() *Mode { return &Mode{pendingGoto: -1} }

// State reports the current modal state.
func (m *Mode) State() State { return m.state }

// GotoLinePending returns a :N line number (1-based) requested by the last
// HandleCmd call and clears it; negative when no jump is pending.
func (m *Mode) GotoLinePending() int {
	n := m.pendingGoto
	m.pendingGoto = -1
	return n
}

func (m *Mode) cnt() int {
	if m.count == 0 {
		return 1
	}
	return m.count
}

func (m *Mode) resetCount() { m.count = 0 }

// HandleKey implements editorview.Modal: it may mutate doc and returns true
// when the key was consumed. In Normal/Visual every key is consumed; in
// Insert/Replace only the keys that leave the state are consumed, so unbound
// keys fall through to the editorview's insert-mode handling.
func (m *Mode) HandleKey(k input.KeyPress, doc *document.Document) bool {
	switch m.state {
	case Insert:
		if k.Key == input.KeyEscape {
			m.state = Normal
			m.op = opNone
			m.resetCount()
			return true
		}
		return false
	case Replace:
		if k.Key == input.KeyEscape {
			m.state = Normal
			return true
		}
		if k.Key == input.KeyNone && k.Rune != 0 && k.Mod == 0 {
			m.replaceOne(doc, k.Rune)
			return true
		}
		return false
	case Visual, VisualLine:
		return m.handleVisual(k, doc)
	case Normal:
		return m.handleNormal(k, doc)
	}
	return true
}

func (m *Mode) handleNormal(k input.KeyPress, doc *document.Document) bool {
	// Numeric prefix builds a repeat count (a lone 0 is Home, handled below).
	if k.Key == input.KeyNone && k.Rune >= '1' && k.Rune <= '9' && k.Mod == 0 {
		m.count = m.count*10 + int(k.Rune-'0')
		return true
	}
	if m.haveReplace {
		m.haveReplace = false
		m.replaceOne(doc, k.Rune)
		return true
	}
	if k.Key == input.KeyEscape {
		m.op = opNone
		m.resetCount()
		return true
	}
	if m.op != opNone {
		return m.finishOperator(k, doc)
	}
	if k.Key == input.KeyNone && k.Mod == 0 {
		switch k.Rune {
		case 'h', 'H':
			doc.Move(document.MoveLeft, false)
		case 'l', 'L':
			doc.Move(document.MoveRight, false)
		case 'j', 'J':
			doc.MoveLines(m.cnt(), false)
		case 'k', 'K':
			doc.MoveLines(-m.cnt(), false)
		case 'w', 'W':
			doc.Move(document.MoveWordRight, false)
		case 'b', 'B':
			doc.Move(document.MoveWordLeft, false)
		case 'e', 'E':
			doc.Move(document.MoveWordRight, false)
		case '0':
			doc.Move(document.MoveHome, false)
		case '^':
			doc.Move(document.MoveHome, false)
		case '$':
			doc.Move(document.MoveEnd, false)
		case 'G':
			doc.Move(document.MoveDocEnd, false)
		case 'g':
			doc.Move(document.MoveDocStart, false)
		case 'i', 'I':
			if k.Rune == 'I' {
				doc.Move(document.MoveHome, false)
			}
			m.state = Insert
		case 'a', 'A':
			if k.Rune == 'A' {
				doc.Move(document.MoveEnd, false)
			} else {
				doc.Move(document.MoveRight, false)
			}
			m.state = Insert
		case 'o':
			doc.SelectNone()
			doc.InsertNewline()
			m.state = Insert
		case 'O':
			doc.SelectNone()
			doc.SetCursor(document.Pos{Line: doc.Cursor().Line})
			doc.InsertNewline()
			m.state = Insert
		case 'x', 'X':
			m.deleteChar(doc)
		case 'D':
			m.opMotion(doc, opDelete, document.MoveEnd)
		case 'C':
			m.opMotion(doc, opChange, document.MoveEnd)
		case 'd':
			m.op = opDelete
		case 'c':
			m.op = opChange
		case 'y':
			m.op = opYank
		case 'p':
			m.pasteAfter(doc)
		case 'P':
			m.pasteBefore(doc)
		case 'r':
			m.haveReplace = true
		case 'v':
			m.state = Visual
			doc.SelectNone()
		case 'V':
			m.state = VisualLine
			doc.SelectNone()
		case 'u':
			doc.Undo()
		case 'R':
			doc.Redo()
		default:
			return true
		}
		m.resetCount()
		return true
	}
	switch k.Key {
	case input.KeyLeft:
		doc.Move(document.MoveLeft, false)
	case input.KeyRight:
		doc.Move(document.MoveRight, false)
	case input.KeyDown:
		doc.MoveLines(m.cnt(), false)
	case input.KeyUp:
		doc.MoveLines(-m.cnt(), false)
	case input.KeyHome:
		doc.Move(document.MoveHome, false)
	case input.KeyEnd:
		doc.Move(document.MoveEnd, false)
	default:
		return true
	}
	m.resetCount()
	return true
}

func (m *Mode) finishOperator(k input.KeyPress, doc *document.Document) bool {
	op := m.op
	count := m.cnt()
	repeat := func(r rune) bool {
		return k.Key == input.KeyNone && k.Rune == r && k.Mod == 0
	}
	switch {
	case repeat('d') || repeat('c') || repeat('y'):
		m.operateLine(doc, op)
	case repeat('w') || repeat('e'):
		m.opMotion(doc, op, document.MoveWordRight)
	case repeat('b'):
		m.opMotion(doc, op, document.MoveWordLeft)
	case repeat('$'):
		m.opMotion(doc, op, document.MoveEnd)
	case repeat('0'):
		m.opMotion(doc, op, document.MoveHome)
	case k.Key == input.KeyLeft:
		m.opMotion(doc, op, document.MoveLeft)
	case k.Key == input.KeyRight:
		m.opMotion(doc, op, document.MoveRight)
	case k.Key == input.KeyDown:
		doc.MoveLines(count, true)
		m.applyOp(doc, op)
	case k.Key == input.KeyUp:
		doc.MoveLines(-count, true)
		m.applyOp(doc, op)
	}
	m.op = opNone
	m.resetCount()
	return true
}

// opMotion starts a selection at the cursor and extends it by dir, then runs
// the pending operator over the selection.
func (m *Mode) opMotion(doc *document.Document, op operator, dir document.MoveDir) {
	doc.SelectNone()
	doc.Move(dir, true)
	m.applyOp(doc, op)
}

func (m *Mode) applyOp(doc *document.Document, op operator) {
	s, e, ok := doc.SelByteOffsets()
	if !ok {
		return
	}
	switch op {
	case opDelete:
		doc.ReplaceRange(s, e, "")
	case opChange:
		doc.ReplaceRange(s, e, "")
		m.state = Insert
	case opYank:
		if t := doc.SelectionText(); t != "" {
			m.lastYank = t
		}
		doc.SelectNone()
	}
}

// operateLine runs the operator over count whole lines starting at the cursor.
func (m *Mode) operateLine(doc *document.Document, op operator) {
	count := m.cnt()
	cur := doc.Cursor()
	start := document.Pos{Line: cur.Line}
	endLine := cur.Line + count
	if endLine > doc.LineCount() {
		endLine = doc.LineCount()
	}
	s := doc.Buffer().PosToOffset(start)
	var eoff int
	if endLine >= doc.LineCount() {
		eoff = doc.Buffer().Len()
	} else {
		eoff = doc.Buffer().PosToOffset(document.Pos{Line: endLine})
	}
	switch op {
	case opDelete:
		doc.ReplaceRange(s, eoff, "")
	case opChange:
		doc.ReplaceRange(s, eoff, "")
		doc.SetCursor(start)
		m.state = Insert
	case opYank:
		if t := doc.Buffer().Text()[s:eoff]; t != "" {
			m.lastYank = t
		}
		doc.SelectNone()
	}
}

func (m *Mode) deleteChar(doc *document.Document) {
	cur := doc.Cursor()
	if cur.Col < doc.Buffer().LineLen(cur.Line) {
		doc.DeleteForward()
	} else {
		doc.Backspace()
	}
}

func (m *Mode) replaceOne(doc *document.Document, r rune) {
	cur := doc.Cursor()
	if cur.Col >= doc.Buffer().LineLen(cur.Line) {
		return
	}
	s := doc.Buffer().PosToOffset(cur)
	e := doc.Buffer().PosToOffset(document.Pos{Line: cur.Line, Col: cur.Col + 1})
	doc.ReplaceRange(s, e, string(r))
}

func (m *Mode) pasteAfter(doc *document.Document) {
	if m.lastYank == "" {
		return
	}
	doc.SelectNone()
	doc.InsertText(m.lastYank)
}

func (m *Mode) pasteBefore(doc *document.Document) {
	if m.lastYank == "" {
		return
	}
	doc.SelectNone()
	doc.InsertText(m.lastYank)
}

func (m *Mode) handleVisual(k input.KeyPress, doc *document.Document) bool {
	if k.Key == input.KeyEscape {
		m.state = Normal
		doc.SelectNone()
		m.resetCount()
		return true
	}
	if k.Key == input.KeyNone && k.Mod == 0 {
		switch k.Rune {
		case 'h', 'H':
			doc.Move(document.MoveLeft, true)
		case 'l', 'L':
			doc.Move(document.MoveRight, true)
		case 'j', 'J':
			doc.MoveLines(m.cnt(), true)
		case 'k', 'K':
			doc.MoveLines(-m.cnt(), true)
		case 'w', 'W':
			doc.Move(document.MoveWordRight, true)
		case 'b', 'B':
			doc.Move(document.MoveWordLeft, true)
		case '$':
			doc.Move(document.MoveEnd, true)
		case '0':
			doc.Move(document.MoveHome, true)
		case 'd', 'x', 'X':
			m.cutVisual(doc)
			m.state = Normal
			m.resetCount()
			return true
		case 'c':
			m.cutVisual(doc)
			m.state = Insert
			m.resetCount()
			return true
		case 'y':
			m.yankVisual(doc)
			m.state = Normal
			m.resetCount()
			return true
		default:
			m.resetCount()
			return true
		}
		m.resetCount()
		return true
	}
	switch k.Key {
	case input.KeyLeft:
		doc.Move(document.MoveLeft, true)
	case input.KeyRight:
		doc.Move(document.MoveRight, true)
	case input.KeyDown:
		doc.MoveLines(m.cnt(), true)
	case input.KeyUp:
		doc.MoveLines(-m.cnt(), true)
	case input.KeyHome:
		doc.Move(document.MoveHome, true)
	case input.KeyEnd:
		doc.Move(document.MoveEnd, true)
	default:
		m.resetCount()
		return true
	}
	m.resetCount()
	return true
}

func (m *Mode) cutVisual(doc *document.Document) {
	s, e, ok := doc.SelByteOffsets()
	if !ok {
		return
	}
	if t := doc.SelectionText(); t != "" {
		m.lastYank = t
	}
	doc.ReplaceRange(s, e, "")
}

func (m *Mode) yankVisual(doc *document.Document) {
	if t := doc.SelectionText(); t != "" {
		m.lastYank = t
	}
	doc.SelectNone()
}

// HandleCmd executes an ex command, see HandleKey docs.
func (m *Mode) HandleCmd(cmd string) (executed, quit, save bool, path string) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return false, false, false, ""
	}
	switch {
	case cmd == "w":
		return true, false, true, ""
	case cmd == "wq", cmd == "x":
		return true, true, true, ""
	case cmd == "q":
		return true, true, false, ""
	case cmd == "q!":
		return true, true, false, ""
	case cmd == "qa", cmd == "qa!":
		return true, true, false, ""
	case strings.HasPrefix(cmd, "w "):
		return true, false, true, strings.TrimSpace(cmd[2:])
	case strings.HasPrefix(cmd, "e "):
		return true, false, false, strings.TrimSpace(cmd[2:])
	}
	if n, err := strconv.Atoi(cmd); err == nil {
		m.pendingGoto = n
		return true, false, false, ""
	}
	return false, false, false, ""
}
