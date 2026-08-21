package textbuf

import (
	"sort"
)

// lineBounds returns the byte range [s, e) of line i's content, excluding the
// terminating '\n'. ok is false for out-of-range lines.
func (b *Buffer) lineBounds(i int) (s, e int, ok bool) {
	if i < 0 || i >= len(b.lineStarts) {
		return 0, 0, false
	}
	s = b.lineStarts[i]
	e = b.Len()
	if i+1 < len(b.lineStarts) {
		e = b.lineStarts[i+1] - 1 // the '\n' itself
	}
	return s, e, true
}

// updateLineIndex repairs the line index after replacing [start, end) with s.
// Edits touching at most one newline are repaired incrementally; anything
// bigger triggers a full single-scan rebuild, which is cheaper than reasoning
// about many shifted boundaries.
func (b *Buffer) updateLineIndex(start, end, delta, removedNL int, s []byte) {
	insNL := 0
	for _, c := range s {
		if c == '\n' {
			insNL++
		}
	}
	switch {
	case insNL == 0 && removedNL == 0:
		shiftStartsAfter(b.lineStarts, start, delta)
	case insNL+removedNL == 1:
		if insNL == 1 {
			j := -1
			for k, c := range s {
				if c == '\n' {
					j = k
					break
				}
			}
			b.lineStarts = insertLineStart(b.lineStarts, start, len(s), start+j+1)
		} else {
			b.lineStarts = removeLineSpan(b.lineStarts, start, end)
		}
	default:
		b.rebuildLineIndex()
	}
}

// shiftStartsAfter moves every line start strictly past off by delta; a
// same-line edit never moves the boundary at or before off.
func shiftStartsAfter(starts []int, off, delta int) {
	for i := sort.Search(len(starts), func(k int) bool { return starts[k] > off }); i < len(starts); i++ {
		starts[i] += delta
	}
}

// insertLineStart updates the index for an insertion of insLen bytes at off
// whose text contains one newline creating the new line start newStart.
// Starts at or before off stay put (the inserted text joins that line);
// starts past off shift right.
func insertLineStart(starts []int, off, insLen, newStart int) []int {
	i := sort.Search(len(starts), func(k int) bool { return starts[k] > off })
	out := make([]int, 0, len(starts)+1)
	out = append(out, starts[:i]...)
	out = append(out, newStart)
	for _, p := range starts[i:] {
		out = append(out, p+insLen)
	}
	return out
}

// removeLineSpan updates the index for a deletion of [start, end) that
// removed exactly one newline: boundaries inside the span vanish and later
// ones shift left. A boundary exactly at start survives (the surviving text
// still begins a line there).
func removeLineSpan(starts []int, start, end int) []int {
	d := end - start
	i := sort.Search(len(starts), func(k int) bool { return starts[k] > start })
	j := sort.Search(len(starts), func(k int) bool { return starts[k] > end })
	out := make([]int, 0, len(starts)-(j-i))
	out = append(out, starts[:i]...)
	for _, p := range starts[j:] {
		out = append(out, p-d)
	}
	return out
}

// rebuildLineIndex recomputes all line starts with one scan over the content.
func (b *Buffer) rebuildLineIndex() {
	starts := make([]int, 1, b.LineCount()+8)
	n := b.Len()
	for i := 0; i < n; i++ {
		if b.byteAt(i) == '\n' {
			starts = append(starts, i+1)
		}
	}
	b.lineStarts = starts
}
