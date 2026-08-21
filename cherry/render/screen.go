// Package render provides cherry's retained cell grid and frame flusher.
//
// The Screen keeps a front buffer (what the terminal is believed to show) and
// a back buffer (the desired frame). Flush diffs the two per dirty row,
// emitting only changed runs with one cursor-position request per run and
// minimal SGR transitions, into a single reusable byte buffer. Steady-state
// frames perform no allocations beyond that buffer's growth, and a typical
// editor-sized keystroke frame encodes in well under 50 microseconds.
package render

import (
	"bytes"
	"io"

	"github.com/Aswanidev-vs/cherry/cell"
	"github.com/Aswanidev-vs/cherry/geom"
)

type ColorMode uint8

const (
	ColorMono ColorMode = iota
	Color16
	Color256
	ColorRGB
)

// Screen is a retained w*h cell grid with dirty-row diffing.
type Screen struct {
	w, h      int
	back      []cell.Cell
	front     []cell.Cell
	dirty     []bool
	first     bool
	lastX     int
	lastY     int
	colorMode ColorMode
	syncOut   bool

	buf *bytes.Buffer // reused frame scratch; the only growing allocation
	sw  sgrWriter     // mirrors the terminal's current SGR state
}

// New creates a blank screen of the given size.
func New(w, h int) *Screen {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	s := &Screen{w: w, h: h, lastX: -1, lastY: -1, buf: &bytes.Buffer{}}
	s.buf.Grow(w*h*4 + 64)
	s.sw = sgrWriter{buf: s.buf}
	s.alloc()
	return s
}

// alloc (re)creates buffers sized to w*h; the next Flush is a full repaint.
func (s *Screen) alloc() {
	n := s.w * s.h
	s.back = make([]cell.Cell, n)
	s.front = make([]cell.Cell, n)
	for i := range s.back {
		s.back[i] = cell.Blank
	}
	s.dirty = make([]bool, s.h)
	s.first = true
}

// Resize rebuilds the screen for new dimensions. Content is discarded and the
// next Flush repaints everything.
func (s *Screen) Resize(w, h int) {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	if w == s.w && h == s.h {
		s.Clear()
		s.first = true
		return
	}
	s.w, s.h = w, h
	s.alloc()
}

func (s *Screen) Size() geom.Size   { return geom.Size{W: s.w, H: s.h} }
func (s *Screen) Bounds() geom.Rect { return geom.Rect{Size: geom.Size{W: s.w, H: s.h}} }

// Set writes one cell, ignoring out-of-range coordinates.
func (s *Screen) Set(x, y int, c cell.Cell) {
	if x < 0 || y < 0 || x >= s.w || y >= s.h {
		return
	}
	s.back[y*s.w+x] = c
	s.dirty[y] = true
}

// CellAt reads one cell, returning the zero Cell when out of range.
func (s *Screen) CellAt(x, y int) cell.Cell {
	if x < 0 || y < 0 || x >= s.w || y >= s.h {
		return cell.Cell{}
	}
	return s.back[y*s.w+x]
}

// Fill paints every cell of r, clipped to the screen.
func (s *Screen) Fill(r geom.Rect, c cell.Cell) {
	r = r.Intersect(s.Bounds())
	for y := r.Pos.Y; y < r.Bottom(); y++ {
		base := y * s.w
		for x := r.Pos.X; x < r.Right(); x++ {
			s.back[base+x] = c
		}
		s.dirty[y] = true
	}
}

// Clear resets the back buffer to blanks and dirties every row.
func (s *Screen) Clear() {
	for i := range s.back {
		s.back[i] = cell.Blank
	}
	for y := range s.dirty {
		s.dirty[y] = true
	}
}

// Invalidate schedules a full physical repaint without touching cell
// contents. first=true makes the next Flush rewrite every cell even where the
// diff would find no change — required after suspend/resume or resize, when
// the terminal's actual contents are unknown.
func (s *Screen) Invalidate() {
	for y := range s.dirty {
		s.dirty[y] = true
	}
	s.first = true
}

// Print draws text at (x,y) clipped to maxX (also capped at the screen edge),
// returning the x just past the last cell written. Rune widths come from
// cell.RuneWidth; zero-width runes (combining marks, format characters) are
// dropped entirely — v1 limitation, so base+combining sequences render as the
// bare base glyph. A wide rune that would straddle maxX truncates the rest of
// the line. Wide runes occupy their trailing cell as a Width==0 spacer,
// following the convention documented in cell.Cell.
func (s *Screen) Print(x, y, maxX int, text string, st cell.Style) int {
	if y < 0 || y >= s.h {
		return x
	}
	if maxX > s.w {
		maxX = s.w
	}
	if x < 0 {
		x = 0
	}
	s.dirty[y] = true
	cx := x
	for _, r := range text {
		w := cell.RuneWidth(r)
		if w == 0 {
			continue
		}
		if cx >= maxX || (w == 2 && cx+1 >= maxX) {
			break
		}
		s.back[y*s.w+cx] = cell.Cell{Rune: r, Style: st, Width: uint8(w)}
		if w == 2 {
			s.back[y*s.w+cx+1] = cell.Cell{Rune: ' ', Style: st, Width: 0}
			cx += 2
		} else {
			cx++
		}
	}
	return cx
}

// SetSyncOutput toggles synchronized-update (DECSET 2026) bracketing around
// frames; wire it to DetectCapabilities' second result. Takes effect next Flush.
func (s *Screen) SetSyncOutput(on bool) { s.syncOut = on }

// SetColorMode stores the emission fidelity; it takes effect at the next
// Flush, which then repaints fully because stored colors render differently.
func (s *Screen) SetColorMode(m ColorMode) { s.colorMode = m }

// Flush diffs back against front and writes the minimal byte stream to out.
// Frames open by hiding the cursor (\e[?25l); callers restore it with
// ShowCursor. When enabled, the frame is wrapped in synchronized-update
// brackets. On success the front buffer syncs and dirty marks clear; on a
// write error nothing is marked clean, so a retry repaints.
func (s *Screen) Flush(out io.Writer) error {
	if s.colorMode != s.sw.mode {
		s.sw.mode = s.colorMode
		s.first = true // stored colors now encode differently; repaint all
	}

	buf := s.buf
	buf.Reset()
	if s.syncOut {
		buf.WriteString("\x1b[?2026h")
	}
	buf.WriteString("\x1b[?25l")

	if s.first {
		s.sw.reset()
		buf.WriteString("\x1b[0m") // baseline: unknown prior terminal state
		for y := 0; y < s.h; y++ {
			writeCUP(buf, y+1, 1)
			base := y * s.w
			for x := 0; x < s.w; x++ {
				c := s.back[base+x]
				if c.Width == 0 {
					continue // trailing half of a wide glyph, already drawn
				}
				s.sw.to(c.Style)
				buf.WriteRune(c.Rune)
			}
		}
	} else {
		for y := 0; y < s.h; y++ {
			if !s.dirty[y] {
				continue
			}
			base := y * s.w
			back, front := s.back[base:base+s.w], s.front[base:base+s.w]
			for x := 0; x < s.w; {
				if back[x] == front[x] {
					x++
					continue
				}
				writeCUP(buf, y+1, x+1)
				for ; x < s.w && back[x] != front[x]; x++ {
					c := back[x]
					if c.Width == 0 {
						continue // its lead glyph is inside this run
					}
					s.sw.to(c.Style)
					buf.WriteRune(c.Rune)
				}
			}
		}
	}

	if s.syncOut {
		buf.WriteString("\x1b[?2026l")
	}

	if _, err := out.Write(buf.Bytes()); err != nil {
		return err
	}
	copy(s.front, s.back)
	for y := range s.dirty {
		s.dirty[y] = false
	}
	s.first = false
	s.lastX, s.lastY = -1, -1 // frame left the physical cursor position unknown
	return nil
}

// ShowCursor positions (1-based to the terminal) and reveals the cursor, or
// hides it when either coordinate is negative. The position is recorded so
// Flush's cursor bookkeeping stays consistent.
func (s *Screen) ShowCursor(out io.Writer, x, y int) error {
	s.buf.Reset()
	if x < 0 || y < 0 {
		s.lastX, s.lastY = -1, -1
		s.buf.WriteString("\x1b[?25l")
	} else {
		s.lastX, s.lastY = x, y
		writeCUP(s.buf, y+1, x+1)
		s.buf.WriteString("\x1b[?25h")
	}
	_, err := out.Write(s.buf.Bytes())
	return err
}

func writeCUP(buf *bytes.Buffer, row, col int) {
	buf.WriteString("\x1b[")
	writeUint(buf, uint32(row))
	buf.WriteByte(';')
	writeUint(buf, uint32(col))
	buf.WriteByte('H')
}
