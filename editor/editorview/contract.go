// Package editorview is the main editing surface: one viewport painting a
// Document with line numbers, syntax spans, selection and cursor emission,
// and handling the full insert-mode key surface (typing, movement,
// selection, clipboard, mouse). A Modal layer may be installed ahead of the
// insert-mode handling (the future vim mode).
package editorview

import (
	"github.com/Aswanidev-vs/cherry"
	"github.com/Aswanidev-vs/cherry/geom"
	"github.com/Aswanidev-vs/cherry/input"
	"github.com/Aswanidev-vs/cherry/widget"

	"github.com/Aswanidev-vs/panda-editor/editor/document"
	"github.com/Aswanidev-vs/panda-editor/editor/highlight"
)

var (
	_ widget.Widget    = (*View)(nil)
	_ widget.Focusable = (*View)(nil)
	_ widget.Cursorer  = (*View)(nil)
)

// Modal intercepts keys before insert-mode handling; returning true
// consumes the key. It never sees mouse, paste or resize events.
type Modal interface {
	HandleKey(k input.KeyPress, doc *document.Document) bool
}

// View implements widget.Widget, widget.Focusable and widget.Cursorer.
type View struct {
	app      *cherry.App
	doc      *document.Document
	modal    Modal
	lexer    *highlight.Lexer
	rows     []lexRow
	states   []highlight.State
	dragging bool
	tabWidth int
	focused  bool
	drawn    bool
	lastRect geom.Rect
}

// New wires a view onto doc; app is used for clipboard glue and may be nil
// in tests (clipboard actions then no-op).
func New(app *cherry.App, doc *document.Document) *View {
	v := &View{app: app, doc: doc, tabWidth: defaultTabWidth}
	v.lexer = highlight.NewLexer(doc.Path())
	return v
}

// SetDocument swaps the document (e.g. opening a file), keeping modal and
// tab settings. It resets selection, keeps the old scroll clamped.
func (v *View) SetDocument(doc *document.Document) {
	v.doc = doc
	v.rows, v.states = nil, nil
	v.dragging = false
	v.lexer = highlight.NewLexer(doc.Path())
	doc.SelectNone()
	doc.SetScrollY(doc.ScrollY())
}

// Document returns the active document.
func (v *View) Document() *document.Document { return v.doc }

// SetModal installs a modal layer ahead of insert-mode keys; nil disables.
func (v *View) SetModal(m Modal) { v.modal = m }

// SetTabWidth controls how tabs render and how Indent aligns; default 4.
func (v *View) SetTabWidth(n int) {
	if n < 1 {
		n = 1
	}
	v.tabWidth = n
}

// Measure fills the available region entirely.
func (v *View) Measure(max geom.Size) geom.Size { return max }

func (v *View) Focus()        { v.focused = true }
func (v *View) Blur()         { v.focused = false }
func (v *View) Focused() bool { return v.focused }
