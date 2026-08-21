package textbuf

import (
	"strings"
	"testing"
)

func TestUndoBurstCoalescing(t *testing.T) {
	b := New()
	for _, r := range "abc" {
		b.InsertRune(Pos{Line: 0, Col: b.LineLen(0)}, r)
	}
	if got := b.Text(); got != "abc" {
		t.Fatalf("Text = %q, want abc", got)
	}
	if !b.Undo() {
		t.Fatal("first Undo should apply")
	}
	if got := b.Text(); got != "" {
		t.Fatalf("after one Undo Text = %q, want empty (whole burst coalesced)", got)
	}
	if b.Undo() {
		t.Fatal("second Undo should report false: burst was one step")
	}
	if !b.Redo() || b.Text() != "abc" {
		t.Fatal("Redo should restore the whole burst")
	}
}

func TestUndoNewlineBreaksTypingRun(t *testing.T) {
	b := New()
	steps := []string{"a", "b", "\n", "c", "d"}
	col := 0
	line := 0
	for _, s := range steps {
		if s == "\n" {
			b.InsertRune(Pos{Line: line, Col: col}, '\n')
			line++
			col = 0
			continue
		}
		b.InsertRune(Pos{Line: line, Col: col}, rune(s[0]))
		col++
	}
	undos := 0
	for b.Undo() {
		undos++
	}
	// "ab" coalesces, the newline stands alone, "cd" coalesces: 3 steps.
	if undos != 3 {
		t.Fatalf("got %d undo steps, want 3 (bursts coalesce; newline breaks the run)", undos)
	}
}

func TestUndoDeleteReversesExactly(t *testing.T) {
	b := FromString("hello world\nsecond line")
	b.DeletePos(Pos{Line: 0, Col: 5}, 6) // remove " world"
	if got := b.Line(0); got != "hello" {
		t.Fatalf("after delete Line(0) = %q, want hello", got)
	}
	snapshot := b.Text()
	if !b.Undo() {
		t.Fatal("Undo should reverse the delete")
	}
	if got := b.Text(); got != "hello world\nsecond line" {
		t.Fatalf("after Undo Text = %q", got)
	}
	if b.Redo() && b.Text() != snapshot {
		t.Fatalf("Redo changed result: %q vs %q", b.Text(), snapshot)
	}
}

// Typing after a plain Insert is a separate step: only InsertRune bursts
// coalesce.
func TestUndoPlainInsertDoesNotCoalesceWithTyping(t *testing.T) {
	b := New()
	b.Insert(0, "base")
	b.InsertRune(Pos{Line: 0, Col: 4}, '!')
	b.Undo() // removes '!'
	if got := b.Text(); got != "base" {
		t.Fatalf("Text = %q, want base", got)
	}
	b.Undo() // removes "base"
	if b.Undo() {
		t.Fatal("only two steps expected")
	}
}

func TestUndoJumpBreaksTypingRun(t *testing.T) {
	b := FromString("....") // 0123... cursor jumps between inserts
	b.InsertRune(Pos{Line: 0, Col: 4}, 'a')
	b.InsertRune(Pos{Line: 0, Col: 0}, 'b') // non-adjacent offset breaks run
	undos := 0
	for b.Undo() {
		undos++
	}
	if undos != 2 {
		t.Fatalf("got %d steps, want 2 (jump splits runs)", undos)
	}
}

func TestUndoNestedGroups(t *testing.T) {
	b := New()
	b.BeginGroup()
	b.Insert(0, "X")
	b.BeginGroup()
	b.Insert(b.Len(), "Y")
	b.EndGroup()
	b.Insert(b.Len(), "Z")
	b.EndGroup()
	if got := b.Text(); got != "XYZ" {
		t.Fatalf("Text = %q, want XYZ", got)
	}
	if !b.Undo() {
		t.Fatal("group should be one step")
	}
	if got := b.Text(); got != "" {
		t.Fatalf("nested group left %q, want empty", got)
	}
	if b.Undo() {
		t.Fatal("nested group must collapse to exactly one step")
	}
	if !b.Redo() || b.Text() != "XYZ" {
		t.Fatal("Redo should replay the whole group")
	}
}

func TestUndoInsideOpenGroupFails(t *testing.T) {
	b := New()
	b.BeginGroup()
	b.Insert(0, "x")
	if b.Undo() || b.Redo() {
		t.Fatal("Undo/Redo must refuse while a group is open")
	}
	b.EndGroup()
	if !b.Undo() {
		t.Fatal("Undo should work again after EndGroup")
	}
}

func TestRedoInvalidationOnFreshEdit(t *testing.T) {
	b := New()
	b.Insert(0, "A")
	b.Undo()
	if !b.Redo() {
		t.Fatal("Redo should work right after Undo")
	}
	b.Undo()
	b.Insert(0, "B") // fresh edit kills redo tail
	if b.Redo() {
		t.Fatal("fresh edit must invalidate redo")
	}
	if got := b.Text(); got != "B" {
		t.Fatalf("Text = %q, want B", got)
	}
}

func TestRedoInvalidatedMidTypingRun(t *testing.T) {
	b := New()
	b.Insert(0, "old")
	b.Undo()
	b.InsertRune(Pos{Line: 0, Col: 0}, 'n')
	b.InsertRune(Pos{Line: 0, Col: 1}, 'e') // continue typing
	if b.Redo() {
		t.Fatal("redo must stay dead during and after a fresh typing run")
	}
	if got := b.Text(); got != "ne" {
		t.Fatalf("Text = %q, want ne", got)
	}
}

func TestUndoDepthCapTrimsOldest(t *testing.T) {
	b := New()
	const total = maxUndoSteps + 88
	for i := 0; i < total; i++ {
		b.Insert(0, "x") // plain Insert: every call its own step
	}
	if b.Len() != total {
		t.Fatalf("Len = %d, want %d", b.Len(), total)
	}
	undos := 0
	for b.Undo() {
		undos++
	}
	if undos != maxUndoSteps {
		t.Fatalf("got %d undo steps, want cap %d", undos, maxUndoSteps)
	}
	if got := b.Len(); got != total-maxUndoSteps {
		t.Fatalf("remaining Len = %d, want %d (oldest steps survive trimming)", got, total-maxUndoSteps)
	}
}

func TestUndoBytesCapDropsOversizedStep(t *testing.T) {
	b := New()
	huge := strings.Repeat("y", maxUndoBytes+1)
	b.Insert(0, huge)
	// A single step larger than the byte budget cannot be retained at all:
	// history is dropped rather than memory exploding.
	if b.Undo() {
		t.Fatal("oversized step should have been trimmed from history")
	}
	if b.Len() != maxUndoBytes+1 {
		t.Fatalf("trimming history must not touch content; Len = %d", b.Len())
	}
}

func TestUndoRestoresModifiedState(t *testing.T) {
	b := New()
	b.Insert(0, "x")
	if !b.Modified() {
		t.Fatal("edit should set modified")
	}
	b.SetModified(false) // simulate save point
	if b.Modified() {
		t.Fatal("should be clean after sync")
	}
	if !b.Undo() {
		t.Fatal("Undo should apply")
	}
	if !b.Modified() {
		t.Fatal("content changed via Undo ⇒ modified")
	}
	if !b.Redo() {
		t.Fatal("Redo should apply")
	}
	if !b.Modified() {
		t.Fatal("content changed via Redo ⇒ modified")
	}
}

func TestGroupOfManyOpsIsOneStep(t *testing.T) {
	b := FromString("aaa\nbbb\nccc")
	b.BeginGroup()
	b.Insert(0, "X")
	b.InsertPos(Pos{Line: 1, Col: 3}, "Y")
	b.DeletePos(Pos{Line: 0, Col: 0}, 1) // removes 'X'
	b.EndGroup()
	want := "aaa\nbbbY\nccc"
	if got := b.Text(); got != want {
		t.Fatalf("setup text = %q, want %q", got, want)
	}
	if !b.Undo() || b.Text() != "aaa\nbbb\nccc" {
		t.Fatalf("one Undo should restore original, got %q", b.Text())
	}
	if b.Undo() {
		t.Fatal("group leaked extra steps")
	}
	if !b.Redo() || b.Text() != want {
		t.Fatalf("Redo of group gave %q, want %q", b.Text(), want)
	}
}

func TestEmptyGroupCommitsNothing(t *testing.T) {
	b := New()
	b.BeginGroup()
	b.BeginGroup()
	b.EndGroup()
	b.EndGroup()
	if b.Undo() {
		t.Fatal("empty nested groups must not create an undo step")
	}
}
