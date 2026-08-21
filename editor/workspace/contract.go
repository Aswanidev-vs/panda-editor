// Package workspace owns the set of open documents, the tab list and the
// single active tab. It is model-only: no rendering. The shell composes its
// state onto the chrome widgets each frame.
package workspace

import (
	"path/filepath"

	"github.com/Aswanidev-vs/panda-editor/editor/document"
)

// tab is one open document plus its presentation flags.
type tab struct {
	doc      *document.Document
	readOnly bool
}

// Workspace is a mutable collection of documents with one active.
type Workspace struct {
	tabs   []tab
	active int
}

// New returns an empty workspace. AddNew() inserts one untitled document
// unless the workspace is empty; shells usually call AddNew only when the
// CLI produced no files.
func New() *Workspace {
	return &Workspace{active: -1}
}

// Add registers doc behind a new tab; the file name (or Untitled) becomes
// the tab label. Returns the zero-based tab index. Add marks the doc
// ReadOnly when readOnly is true so the tab list shows it.
func (w *Workspace) Add(doc *document.Document, readOnly bool) int {
	if readOnly && doc != nil {
		doc.SetReadOnly(true)
	}
	w.tabs = append(w.tabs, tab{doc: doc, readOnly: readOnly})
	w.active = len(w.tabs) - 1
	return w.active
}

// Close removes tab i; the nearest remaining tab becomes active (last one
// wins when closing the tail). I goes out of range on the last tab and
// returns false (the shell treats "cannot close" as "stay alive").
func (w *Workspace) Close(i int) bool {
	if i < 0 || i >= len(w.tabs) || i != w.active {
		return false
	}
	w.tabs = append(w.tabs[:i], w.tabs[i+1:]...)
	if len(w.tabs) == 0 {
		w.active = -1
		return false
	}
	if w.active > len(w.tabs)-1 {
		w.active = len(w.tabs) - 1
	}
	return true
}

// Count / Active report size and active tab index. active < 0 means empty.
func (w *Workspace) Count() int  { return len(w.tabs) }
func (w *Workspace) Active() int { return w.active }

// SetActive switches tabs (clamped to bounds). Does nothing on an empty
// workspace.
func (w *Workspace) SetActive(i int) {
	if len(w.tabs) == 0 {
		return
	}
	if i < 0 {
		i = 0
	}
	if i > len(w.tabs)-1 {
		i = len(w.tabs) - 1
	}
	w.active = i
}

// Doc returns the document of tab i (nil when out of range). ActiveDoc
// returns the active document or nil.
func (w *Workspace) Doc(i int) *document.Document {
	if i < 0 || i >= len(w.tabs) {
		return nil
	}
	return w.tabs[i].doc
}

func (w *Workspace) ActiveDoc() *document.Document { return w.Doc(w.active) }

// TabLabel builds the display label for tab i: file base name (or
// "Untitled"), with a trailing " •" when modified and a leading "RO " for
// read-only tabs.
func (w *Workspace) TabLabel(i int) string {
	if i < 0 || i >= len(w.tabs) {
		return ""
	}
	t := w.tabs[i]
	label := "Untitled"
	if t.doc != nil && t.doc.Path() != "" {
		label = filepath.Base(t.doc.Path())
	}
	if t.doc != nil && t.doc.Modified() {
		label += " •"
	}
	if t.readOnly {
		label = "RO " + label
	}
	return label
}

// FindByPath locates an already-open tab for path (exact match on
// Path()); -1 when none. Shells use it to route duplicate opens to the
// existing tab instead of opening twice.
func (w *Workspace) FindByPath(path string) int {
	for i, t := range w.tabs {
		if t.doc != nil && t.doc.Path() != "" && t.doc.Path() == path {
			return i
		}
	}
	return -1
}

// NewUntitled is a convenience wrapper around New() + New() + Add with a
// fresh empty document.
func (w *Workspace) NewUntitled() int {
	return w.Add(document.New(), false)
}
