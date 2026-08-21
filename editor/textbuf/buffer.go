// Package textbuf is the panda editor's text core: a gap-buffer document
// model with rune-aware positions, an incrementally maintained line index,
// bounded command-pattern undo, and BOM/UTF-16 aware file IO.
//
// Lines are separated by '\n' bytes only. '\r' bytes are ordinary content
// and are never inserted, stripped, or normalized.
package textbuf

import (
	"sort"
	"unicode/utf8"
)

// Pos is a position inside a buffer. Col counts runes, not bytes.
type Pos struct {
	Line int
	Col  int
}

// Buffer holds one text document in a gap buffer of bytes.
//
// A Buffer is NOT safe for concurrent use: callers must confine every method
// call to a single goroutine; there is no internal locking.
type Buffer struct {
	data     []byte // content plus free space; the gap lives at [gapStart, gapEnd)
	gapStart int
	gapEnd   int

	lineStarts []int // byte offset of each line start; element 0 is always 0

	undoStack  []undoEntry
	redoStack  []undoEntry
	undoBytes  int
	group      []cmd // commands accumulated while a group is open
	groupDepth int

	typing typingRun // pending coalesced InsertRune burst, see undo.go

	bom bomKind // encoding marker remembered at Open time (io.go)

	edits      int64 // bumped by every content change, including undo/redo
	savedEdits int64 // value of edits at the last save/load; edits!=saved ⇒ dirty
}

// New returns an empty buffer with a single empty line.
func New() *Buffer { return newFromBytes(nil) }

// FromString returns a buffer holding s verbatim; line endings inside s are
// preserved as-is. The buffer starts out unmodified.
func FromString(s string) *Buffer { return newFromBytes([]byte(s)) }

func newFromBytes(data []byte) *Buffer {
	size := len(data) + 64
	b := &Buffer{
		data:     make([]byte, size),
		gapStart: len(data),
		gapEnd:   size,
	}
	copy(b.data, data)
	b.rebuildLineIndex()
	return b
}

// Len returns the total number of content bytes.
func (b *Buffer) Len() int { return len(b.data) - b.gapLen() }

func (b *Buffer) gapLen() int { return b.gapEnd - b.gapStart }

// moveGap slides the gap so it starts at logical offset to.
func (b *Buffer) moveGap(to int) {
	if to == b.gapStart {
		return
	}
	n := b.gapLen()
	if to < b.gapStart {
		copy(b.data[to+n:b.gapStart+n], b.data[to:b.gapStart])
	} else {
		copy(b.data[b.gapStart:], b.data[b.gapEnd:to+n])
	}
	b.gapStart = to
	b.gapEnd = to + n
}

// growGap makes room for at least need free bytes, doubling capacity when it
// is not already large enough.
func (b *Buffer) growGap(need int) {
	if need <= b.gapLen() {
		return
	}
	tail := len(b.data) - b.gapEnd
	newCap := 2 * len(b.data)
	if min := (b.Len() + need) + need/2; newCap < min {
		newCap = min
	}
	nd := make([]byte, newCap)
	copy(nd, b.data[:b.gapStart])
	copy(nd[newCap-tail:], b.data[b.gapEnd:])
	b.data = nd
	b.gapEnd = newCap - tail
}

// byteAt returns the logical byte at offset i (which must be in range).
func (b *Buffer) byteAt(i int) byte {
	if i >= b.gapStart {
		return b.data[i+b.gapLen()]
	}
	return b.data[i]
}

// slice returns a copy-free view of [start, end) when the range does not
// cross the gap, otherwise materializes the joined bytes.
func (b *Buffer) slice(start, end int) []byte {
	if start >= end {
		return nil
	}
	g := b.gapLen()
	if end <= b.gapStart {
		return b.data[start:end]
	}
	if start >= b.gapStart {
		return b.data[start+g : end+g]
	}
	out := make([]byte, end-start)
	copy(out, b.data[start:b.gapStart])
	copy(out[b.gapStart-start:], b.data[b.gapEnd:end+g])
	return out
}

// Text returns the entire buffer contents.
func (b *Buffer) Text() string {
	n := b.Len()
	b.moveGap(n) // seek to end so the document is one contiguous run
	return string(b.data[:n])
}

// LineCount returns the number of lines; an empty buffer has one empty line.
func (b *Buffer) LineCount() int { return len(b.lineStarts) }

// Line returns line i without its trailing '\n' ('\r' stays, it is content).
// Out-of-range i yields "".
func (b *Buffer) Line(i int) string {
	s, e, ok := b.lineBounds(i)
	if !ok {
		return ""
	}
	return string(b.slice(s, e))
}

// LineLen returns the length of line i in runes; out-of-range i yields 0.
func (b *Buffer) LineLen(i int) int {
	s, e, ok := b.lineBounds(i)
	if !ok {
		return 0
	}
	n := 0
	for off := s; off < e; off++ {
		if b.byteAt(off)&0xC0 != 0x80 { // same counting rule as utf8.RuneCount
			n++
		}
	}
	return n
}

// Insert inserts s at byte offset off, clamped to [0, Len].
func (b *Buffer) Insert(off int, s string) {
	b.insert(off, []byte(s), false)
}

// InsertPos inserts s at rune-column position p (clamped into range).
func (b *Buffer) InsertPos(p Pos, s string) {
	b.insert(b.PosToOffset(p), []byte(s), false)
}

// InsertRune inserts a single rune at p. If the target offset falls inside a
// multi-byte rune it snaps outward onto a rune boundary first. Consecutive
// calls on the same line coalesce into one undo step until the run breaks
// (see undo.go).
func (b *Buffer) InsertRune(p Pos, r rune) {
	off := b.snapBack(b.PosToOffset(p))
	b.insert(off, []byte(string(r)), true)
}

// insert is the common path for all insertions; typing marks inserts that may
// extend an automatic typing-coalescing run. A newline never joins a run.
func (b *Buffer) insert(off int, s []byte, typing bool) {
	if len(s) == 0 {
		return
	}
	off = clampInt(off, 0, b.Len())
	isNL := len(s) == 1 && s[0] == '\n'
	extend := typing && !isNL && b.typing.active && off == b.typing.end
	if !extend {
		b.flushTyping()
	}
	b.clearRedo()
	switch {
	case extend:
		b.typing.buf = append(b.typing.buf, s...)
		b.typing.end = off + len(s)
	case typing && !isNL:
		b.typing = typingRun{active: true, start: off, end: off + len(s), buf: append([]byte(nil), s...)}
	default:
		b.record(cmd{start: off, inserted: append([]byte(nil), s...)})
	}
	b.splice(off, off, s)
	b.edits++
}

// DeleteRange removes the bytes in [start, end); both ends are clamped to the
// document and swapped if reversed.
func (b *Buffer) DeleteRange(start, end int) {
	if start > end {
		start, end = end, start
	}
	n := b.Len()
	start = clampInt(start, 0, n)
	end = clampInt(end, 0, n)
	if start == end {
		return
	}
	b.flushTyping()
	b.clearRedo()
	// Copy: slice may alias the gap buffer, and splice is about to clobber
	// exactly these bytes.
	b.record(cmd{start: start, removed: append([]byte(nil), b.slice(start, end)...)})
	b.splice(start, end, nil)
	b.edits++
}

// DeletePos deletes up to n runes forward from p (a '\n' counts as one rune).
func (b *Buffer) DeletePos(p Pos, n int) {
	if n <= 0 {
		return
	}
	start := b.PosToOffset(p)
	b.DeleteRange(start, b.advanceRunes(start, n))
}

// PosToOffset converts p to a byte offset, clamping line and column into
// range. The result always lands on a rune boundary of well-formed data.
func (b *Buffer) PosToOffset(p Pos) int {
	line := clampInt(p.Line, 0, b.LineCount()-1)
	s, e, _ := b.lineBounds(line)
	lb := b.slice(s, e)
	off := 0
	for col := 0; col < p.Col && off < len(lb); col++ {
		_, sz := utf8.DecodeRune(lb[off:])
		off += sz
	}
	if off > len(lb) {
		off = len(lb)
	}
	return s + off
}

// OffsetToPos converts a byte offset to a Pos, clamping the offset and
// snapping mid-rune offsets back onto the containing rune's start. Column
// counting steps whole runes via DecodeRune so malformed bytes stay
// consistent with PosToOffset/advanceRunes (one step per bad byte).
func (b *Buffer) OffsetToPos(off int) Pos {
	off = b.snapBack(clampInt(off, 0, b.Len()))
	line := sort.Search(len(b.lineStarts), func(i int) bool { return b.lineStarts[i] > off }) - 1
	s, _, _ := b.lineBounds(line)
	col := 0
	for i := s; i < off; {
		var buf [utf8.UTFMax]byte
		buf[0] = b.byteAt(i)
		m := 1
		if buf[0] >= utf8.RuneSelf {
			for m < utf8.UTFMax && i+m < off {
				buf[m] = b.byteAt(i + m)
				m++
			}
		}
		_, sz := utf8.DecodeRune(buf[:m])
		if sz < 1 {
			sz = 1
		}
		i += sz
		col++
	}
	return Pos{Line: line, Col: col}
}

// snapBack moves off left onto the start of the rune containing it: the
// nearest position at or before off whose byte is not a UTF-8 continuation.
// Position Len counts as a boundary.
func (b *Buffer) snapBack(off int) int {
	for off > 0 && off < b.Len() && b.byteAt(off)&0xC0 == 0x80 {
		off--
	}
	return off
}

// advanceRunes returns the offset n whole runes after off, clamped to Len.
func (b *Buffer) advanceRunes(off, n int) int {
	total := b.Len()
	for n > 0 && off < total {
		var buf [utf8.UTFMax]byte
		buf[0] = b.byteAt(off)
		m := 1
		if buf[0] >= utf8.RuneSelf {
			for m < utf8.UTFMax && off+m < total {
				buf[m] = b.byteAt(off + m)
				m++
			}
		}
		if _, sz := utf8.DecodeRune(buf[:m]); sz > 0 {
			off += sz
		} else {
			off++
		}
		n--
	}
	return off
}

// splice replaces the byte range [start, end) with s and maintains the line
// index. Callers must have recorded any undo information first: removed bytes
// are gone by the time this returns.
func (b *Buffer) splice(start, end int, s []byte) {
	removedNL := 0
	for i := start; i < end; i++ {
		if b.byteAt(i) == '\n' {
			removedNL++
		}
	}
	if end > start {
		b.moveGap(start)
		b.gapEnd += end - start
	}
	if len(s) > 0 {
		b.moveGap(start)
		if b.gapLen() < len(s) {
			b.growGap(len(s))
		}
		copy(b.data[start:], s)
		b.gapStart += len(s)
	}
	b.updateLineIndex(start, end, len(s)-(end-start), removedNL, s)
}

// Modified reports whether the content differs from its last saved state.
func (b *Buffer) Modified() bool { return b.edits != b.savedEdits }

// SetModified overrides the dirty flag: false re-syncs the save point to the
// current state, true marks the buffer dirty until the next save or sync.
func (b *Buffer) SetModified(v bool) {
	if v {
		b.savedEdits = b.edits - 1
		return
	}
	b.savedEdits = b.edits
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
