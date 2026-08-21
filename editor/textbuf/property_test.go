package textbuf

import (
	"math/rand"
	"strings"
	"testing"
	"unicode/utf8"
)

// oracle is a naive []byte mirror of the buffer; every check compares the gap
// buffer against plain byte slicing and strings.Split semantics.
type oracle struct {
	data    []byte
	history [][]byte // snapshots taken before each op, for undo verification
}

func (o *oracle) line(i int) string {
	parts := strings.Split(string(o.data), "\n")
	if i < 0 || i >= len(parts) {
		return ""
	}
	return parts[i]
}

func (o *oracle) offsetToPos(off int) Pos {
	if off > len(o.data) {
		off = len(o.data)
	}
	for off > 0 && off < len(o.data) && o.data[off]&0xC0 == 0x80 { // snap back to rune start
		off--
	}
	line := 1 + strings.Count(string(o.data[:off]), "\n")
	lineStart := 0
	if i := strings.LastIndexByte(string(o.data[:off]), '\n'); i >= 0 {
		lineStart = i + 1
	}
	col := utf8.RuneCount(o.data[lineStart:off])
	return Pos{Line: line - 1, Col: col}
}

var fragPool = []string{"a", "Z", "9", "\n", "\r", " ", "日", "😀", "xyz", "ok\r\n"}

func randomFragment(rng *rand.Rand) string {
	n := 1 + rng.Intn(3)
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteString(fragPool[rng.Intn(len(fragPool))])
	}
	return sb.String()
}

func cloneBytes(b []byte) []byte {
	out := make([]byte, len(b))
	copy(out, b)
	return out
}

// TestPropertyAgainstOracle drives ~500 seeded-random edits into both the
// buffer and a naive byte slice, comparing full text, line counts, random
// lines, offset probes, and interleaved undo steps after every operation.
func TestPropertyAgainstOracle(t *testing.T) {
	rng := rand.New(rand.NewSource(20240611))
	o := &oracle{}
	b := New()

	check := func(stage string) {
		t.Helper()
		if got := b.Text(); got != string(o.data) {
			t.Fatalf("%s: Text = %q, want %q", stage, got, o.data)
		}
		if b.Len() != len(o.data) {
			t.Fatalf("%s: Len = %d, want %d", stage, b.Len(), len(o.data))
		}
		wantLines := strings.Count(string(o.data), "\n") + 1
		if b.LineCount() != wantLines {
			t.Fatalf("%s: LineCount = %d, want %d", stage, b.LineCount(), wantLines)
		}
		for k := 0; k < 3; k++ {
			i := rng.Intn(wantLines+3) - 1 // includes out-of-range probes
			if got := b.Line(i); got != o.line(i) {
				t.Fatalf("%s: Line(%d) = %q, want %q", stage, i, got, o.line(i))
			}
		}
		for k := 0; k < 3; k++ {
			off := rng.Intn(b.Len() + 2) // may exceed end; both sides clamp
			gotPos := b.OffsetToPos(off)
			wantPos := o.offsetToPos(off)
			if gotPos != wantPos {
				t.Fatalf("%s: OffsetToPos(%d) = %+v, want %+v (text %q)", stage, off, gotPos, wantPos, o.data)
			}
			if back := b.PosToOffset(gotPos); back != b.snapBack(minInt(off, b.Len())) {
				t.Fatalf("%s: PosToOffset roundtrip of %d gave %d", stage, off, back)
			}
		}
	}

	const iterations = 500
	for iter := 0; iter < iterations; iter++ {
		switch roll := rng.Intn(10); {
		case roll < 6: // insert
			off := rng.Intn(len(o.data) + 1)
			frag := randomFragment(rng)
			o.history = append(o.history, cloneBytes(o.data))
			o.data = append(o.data[:off], append([]byte(frag), o.data[off:]...)...)
			b.Insert(off, frag)
		case roll < 9: // delete
			i, j := rng.Intn(len(o.data)+1), rng.Intn(len(o.data)+1)
			if i > j {
				i, j = j, i
			}
			if i == j {
				break // no-op by design on both sides
			}
			o.history = append(o.history, cloneBytes(o.data))
			o.data = append(o.data[:i], o.data[j:]...)
			b.DeleteRange(i, j)
		default: // occasional undo against recorded history
			if len(o.history) > 0 && rng.Intn(2) == 0 {
				if !b.Undo() {
					t.Fatalf("iter %d: Undo returned false with history pending", iter)
				}
				o.data = o.history[len(o.history)-1]
				o.history = o.history[:len(o.history)-1]
			}
		}
		check("after iter")
		if iter%50 == 49 && !b.Modified() {
			t.Fatalf("iter %d: buffer edited but not modified", iter)
		}
	}

	// Drain the remaining history through Undo; contents must match exactly.
	for len(o.history) > 0 {
		if !b.Undo() {
			t.Fatal("Undo ended before oracle history drained")
		}
		o.data = o.history[len(o.history)-1]
		o.history = o.history[:len(o.history)-1]
	}
	check("after full drain")
	if b.Undo() {
		t.Fatal("Undo should report false once history is exhausted")
	}
}

func minInt(a, c int) int {
	if a < c {
		return a
	}
	return c
}

// TestPropertyTypingCoalesceUnderSeededRandom hammers InsertRune bursts,
// newline inserts, and deletes against a coalescing model: a burst is ONE
// pending step while consecutive bursts land adjacent, and anything else
// (newline, delete, undo) materializes it.
func TestPropertyTypingCoalesceUnderSeededRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(77))
	b := New()
	expectedSteps := 0 // finished steps currently on the undo stack
	pending := false   // a typing run is active and not yet flushed
	runEnd := Pos{}    // where an extending burst must continue

	for i := 0; i < 300; i++ {
		switch roll := rng.Intn(5); {
		case roll < 2: // typing burst of ASCII letters on one line
			burstLen := 1 + rng.Intn(8)
			line := rng.Intn(b.LineCount())
			col := rng.Intn(b.LineLen(line) + 1)
			at := Pos{Line: line, Col: col}
			if !(pending && at == runEnd) {
				if pending {
					expectedSteps++ // previous run flushed by the break
				}
				pending = true
			}
			for k := 0; k < burstLen; k++ {
				b.InsertRune(at, rune('a'+rng.Intn(4)))
				at.Col++
			}
			runEnd = at
		case roll == 2: // newline breaks any run and adds its own step
			if pending {
				expectedSteps++
				pending = false
			}
			line := rng.Intn(b.LineCount())
			b.InsertRune(Pos{Line: line, Col: rng.Intn(b.LineLen(line) + 1)}, '\n')
			expectedSteps++
		case roll == 3: // delete breaks runs and adds its own step
			if pending {
				expectedSteps++
				pending = false
			}
			line := rng.Intn(b.LineCount())
			col := rng.Intn(b.LineLen(line) + 1)
			// A delete anchored at the very end of the document removes
			// nothing and therefore records no step.
			if !(line == b.LineCount()-1 && col == b.LineLen(line)) {
				b.DeletePos(Pos{Line: line, Col: col}, 1+rng.Intn(3))
				expectedSteps++
			}
		default: // undo drains everything that exists, including pending
			if pending {
				expectedSteps++
				pending = false
			}
			drained := 0
			for b.Undo() {
				drained++
			}
			if drained != expectedSteps {
				t.Fatalf("iter %d: drained %d steps, want %d", i, drained, expectedSteps)
			}
			expectedSteps = 0
		}
	}
	if pending {
		expectedSteps++
	}
	drained := 0
	for b.Undo() {
		drained++
	}
	if drained != expectedSteps {
		t.Fatalf("final drain got %d steps, want %d", drained, expectedSteps)
	}
}
