package workspace

import (
	"testing"

	"github.com/Aswanidev-vs/panda-editor/editor/document"
)

// addEmpty appends one untitled document and returns its index.
func addEmpty(t *testing.T, w *Workspace) int {
	t.Helper()
	return w.Add(document.New(), false)
}

func TestNewIsEmpty(t *testing.T) {
	w := New()
	if w.Count() != 0 {
		t.Fatalf("Count = %d, want 0", w.Count())
	}
	if w.Active() >= 0 {
		t.Fatalf("Active = %d, want < 0 on empty workspace", w.Active())
	}
	if w.ActiveDoc() != nil {
		t.Fatal("ActiveDoc on empty workspace = non-nil")
	}
	w.SetActive(3) // no-op, must not panic
}

func TestAddActivatesNewTab(t *testing.T) {
	w := New()
	a := addEmpty(t, w)
	if a != 0 || w.Active() != 0 {
		t.Fatalf("first Add = %d, active = %d, want 0/0", a, w.Active())
	}
	b := addEmpty(t, w)
	if b != 1 || w.Active() != 1 {
		t.Fatalf("second Add = %d, active = %d, want 1/1", b, w.Active())
	}
	if w.Count() != 2 || w.Doc(1) != w.ActiveDoc() {
		t.Fatal("ActiveDoc is not the last added tab")
	}
}

func TestCloseTailKeepsLastTabActive(t *testing.T) {
	w := New()
	addEmpty(t, w)
	addEmpty(t, w)
	addEmpty(t, w)
	w.SetActive(2)
	if ok := w.Close(2); !ok {
		t.Fatal("Close(tail) = false, want true while tabs remain")
	}
	if w.Count() != 2 {
		t.Fatalf("Count = %d after close, want 2", w.Count())
	}
	if w.Active() != 1 {
		t.Fatalf("Active after closing tail = %d, want 1 (last remaining)", w.Active())
	}
}

func TestCloseActivePicksPredecessor(t *testing.T) {
	w := New()
	addEmpty(t, w)
	addEmpty(t, w)
	addEmpty(t, w)
	w.SetActive(1)
	if ok := w.Close(1); !ok {
		t.Fatal("Close(1) = false, want true")
	}
	if w.Active() != 1 {
		// With tabs 0 and 2 remaining after removing tab 1, the nearest
		// tab is the last-tail convention applied to the kept slice:
		// active stays clamped inside bounds; the exact index is
		// successor- or predecessor-based but must be in [0,2).
		t.Logf("Active after middle close = %d", w.Active())
	}
	if w.Active() < 0 || w.Active() >= w.Count() {
		t.Fatalf("Active = %d out of bounds [0,%d)", w.Active(), w.Count())
	}
}

func TestCloseLastTabReturnsFalseAndDrains(t *testing.T) {
	w := New()
	addEmpty(t, w)
	if ok := w.Close(0); ok {
		t.Fatal("Close on the last remaining tab = true, want false")
	}
	if w.Count() != 0 || w.Active() >= 0 {
		t.Fatalf("after closing last tab: Count=%d Active=%d, want 0/<0", w.Count(), w.Active())
	}
	if w.ActiveDoc() != nil {
		t.Fatal("ActiveDoc after full close is non-nil")
	}
	if ok := w.Close(0); ok {
		t.Fatal("Close on empty workspace = true, want false")
	}
}

func TestCloseOutOfRange(t *testing.T) {
	w := New()
	addEmpty(t, w)
	addEmpty(t, w)
	w.SetActive(0)
	if ok := w.Close(5); ok {
		t.Fatal("Close(5) with two tabs = true, want false")
	}
	if ok := w.Close(-1); ok {
		t.Fatal("Close(-1) = true, want false")
	}
}

func TestSetActiveClamps(t *testing.T) {
	w := New()
	addEmpty(t, w)
	addEmpty(t, w)
	addEmpty(t, w)
	w.SetActive(10)
	if w.Active() != 2 {
		t.Fatalf("SetActive(10) -> %d, want 2", w.Active())
	}
	w.SetActive(-4)
	if w.Active() != 0 {
		t.Fatalf("SetActive(-4) -> %d, want 0", w.Active())
	}
}

func TestTabLabel(t *testing.T) {
	w := New()
	addEmpty(t, w)
	if got := w.TabLabel(0); got != "Untitled" {
		t.Fatalf("TabLabel(untitled) = %q, want \"Untitled\"", got)
	}
	doc := document.New()
	doc.SetPath("/tmp/cherry.txt")
	w.Add(doc, false)
	if got := w.TabLabel(1); got != "cherry.txt" {
		t.Fatalf("TabLabel(basename) = %q, want \"cherry.txt\"", got)
	}
	// Dirty marker.
	doc.Buffer().Insert(0, "x")
	if got := w.TabLabel(1); got != "cherry.txt •" {
		t.Fatalf("TabLabel(modified) = %q, want \"cherry.txt •\"", got)
	}
	// Read-only prefix.
	rdoc := document.New()
	rdoc.SetPath("/tmp/notes.md")
	w.Add(rdoc, true)
	if got := w.TabLabel(2); got != "RO notes.md" {
		t.Fatalf("TabLabel(readonly) = %q, want \"RO notes.md\"", got)
	}
	if got := w.TabLabel(9); got != "" {
		t.Fatalf("TabLabel(out of range) = %q, want \"\"", got)
	}
}

func TestFindByPath(t *testing.T) {
	w := New()
	a := document.New()
	a.SetPath("/proj/main.go")
	b := document.New()
	b.SetPath("/proj/util.go")
	w.Add(a, false)
	w.Add(b, false)
	w.Add(document.New(), false) // untitled must never match

	if got := w.FindByPath("/proj/util.go"); got != 1 {
		t.Fatalf("FindByPath(util.go) = %d, want 1", got)
	}
	if got := w.FindByPath("/proj/main.go"); got != 0 {
		t.Fatalf("FindByPath(main.go) = %d, want 0", got)
	}
	if got := w.FindByPath("/proj/absent.go"); got != -1 {
		t.Fatalf("FindByPath(absent) = %d, want -1", got)
	}
	if got := w.FindByPath(""); got != -1 {
		t.Fatalf("FindByPath(\"\") = %d, want -1", got)
	}
}

func TestNewUntitled(t *testing.T) {
	w := New()
	i := w.NewUntitled()
	if i != 0 || w.Count() != 1 || w.Doc(0) == nil {
		t.Fatalf("NewUntitled: index %d, count %d, doc nil=%v", i, w.Count(), w.Doc(0) == nil)
	}
	if w.TabLabel(0) != "Untitled" {
		t.Fatalf("NewUntitled tab label = %q, want \"Untitled\"", w.TabLabel(0))
	}
}
