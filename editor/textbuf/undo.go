package textbuf

const (
	maxUndoSteps = 512
	maxUndoBytes = 32 << 20 // cap on total captured bytes across the undo stack
)

// cmd is one reversible replacement: bytes [start, start+len(removed)) were
// replaced by inserted (either side may be empty). Both directions are
// computable from these three fields, so memory scales with edits rather
// than document size.
type cmd struct {
	start    int
	removed  []byte
	inserted []byte
}

// undoEntry is a single undo step: one command, or an entire (possibly
// nested) group collapsed into an ordered command list.
type undoEntry struct {
	cmds  []cmd
	bytes int // len(removed)+len(inserted) summed over cmds, for capping
}

// typingRun holds an in-progress coalesced InsertRune burst. The runes stay
// out of the undo stack until the run breaks, then flush as one step.
type typingRun struct {
	active bool
	start  int
	end    int // offset just past the last inserted rune
	buf    []byte
}

// BeginGroup opens an undo group: commands issued until the matching EndGroup
// collapse into a single undo step. Groups nest. Any active typing run is
// flushed first so bursts never straddle a group boundary.
func (b *Buffer) BeginGroup() {
	b.flushTyping()
	b.groupDepth++
}

// EndGroup closes the innermost open group and commits its commands as one
// undo step. Unbalanced calls are ignored.
func (b *Buffer) EndGroup() {
	if b.groupDepth == 0 {
		return
	}
	b.groupDepth--
	if b.groupDepth > 0 {
		return
	}
	e := undoEntry{cmds: b.group}
	for _, c := range e.cmds {
		e.bytes += len(c.removed) + len(c.inserted)
	}
	b.group = nil
	if len(e.cmds) > 0 {
		b.pushUndo(e)
	}
}

// Undo reverts the most recent step, returning false when history is
// exhausted or a group is still open.
func (b *Buffer) Undo() bool {
	if b.groupDepth > 0 {
		return false
	}
	b.flushTyping()
	if len(b.undoStack) == 0 {
		return false
	}
	e := b.undoStack[len(b.undoStack)-1]
	b.undoStack = b.undoStack[:len(b.undoStack)-1]
	b.undoBytes -= e.bytes
	b.apply(e, true)
	b.redoStack = append(b.redoStack, e)
	b.edits++
	return true
}

// Redo reapplies the most recently undone step, returning false when there is
// nothing to redo.
func (b *Buffer) Redo() bool {
	if b.groupDepth > 0 {
		return false
	}
	if len(b.redoStack) == 0 {
		return false
	}
	e := b.redoStack[len(b.redoStack)-1]
	b.redoStack = b.redoStack[:len(b.redoStack)-1]
	b.apply(e, false)
	b.pushUndo(e)
	b.edits++
	return true
}

// apply executes e's commands forward or, for undo, in reverse order.
func (b *Buffer) apply(e undoEntry, backward bool) {
	if backward {
		for i := len(e.cmds) - 1; i >= 0; i-- {
			c := e.cmds[i]
			b.splice(c.start, c.start+len(c.inserted), c.removed)
		}
		return
	}
	for _, c := range e.cmds {
		b.splice(c.start, c.start+len(c.removed), c.inserted)
	}
}

// record queues c as an undoable change, folding it into the open group if
// one exists. It does not touch the content itself.
func (b *Buffer) record(c cmd) {
	if b.groupDepth > 0 {
		b.group = append(b.group, c)
		return
	}
	b.pushUndo(undoEntry{cmds: []cmd{c}, bytes: len(c.removed) + len(c.inserted)})
}

func (b *Buffer) pushUndo(e undoEntry) {
	b.undoStack = append(b.undoStack, e)
	b.undoBytes += e.bytes
	for len(b.undoStack) > maxUndoSteps || b.undoBytes > maxUndoBytes {
		b.undoBytes -= b.undoStack[0].bytes
		b.undoStack = b.undoStack[1:] // trim oldest; content already edited stays
	}
}

// clearRedo drops the redo tail: any fresh edit invalidates it.
func (b *Buffer) clearRedo() {
	b.redoStack = nil
}

// flushTyping materializes an active typing run as an undo command, ending
// the run. Runs break on newline inserts, deletes, groups, cursor jumps away
// from the insertion point, and undo/redo.
func (b *Buffer) flushTyping() {
	if !b.typing.active {
		return
	}
	run := b.typing
	b.typing = typingRun{}
	b.record(cmd{start: run.start, inserted: run.buf})
}
