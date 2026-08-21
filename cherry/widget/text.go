package widget

import (
	"strings"

	"github.com/Aswanidev-vs/cherry/cell"
	"github.com/Aswanidev-vs/cherry/geom"
)

// Align selects horizontal placement of rendered lines.
type Align uint8

const (
	AlignLeft Align = iota
	AlignCenter
	AlignRight
)

// Text renders multi-line content, optionally wrapped and aligned. Lines
// beyond the assigned rect are clipped; long lines are clipped by Print.
type Text struct {
	Base

	Content string
	Style   cell.Style
	Wrap    bool
	Align   Align
}

func (t *Text) Measure(max geom.Size) geom.Size {
	raw := strings.Split(t.Content, "\n")
	w, h := 0, len(raw)
	for _, ln := range raw {
		if lw := strWidth(ln); lw > w {
			w = lw
		}
	}
	// Re-wrap only when the natural width would be squeezed anyway; the
	// wrapped height can then exceed the logical line count.
	if t.Wrap && max.W > 0 && w > max.W {
		lines := WrapText(t.Content, max.W)
		h = len(lines)
		w = 0
		for _, ln := range lines {
			if lw := strWidth(ln); lw > w {
				w = lw
			}
		}
	}
	return fit(geom.Size{W: w, H: h}, max)
}

func (t *Text) Draw(ctx *DrawCtx) {
	r := ctx.Rect
	if r.Empty() {
		return
	}
	for i, ln := range t.lines(r.Size.W) {
		y := r.Pos.Y + i
		if y >= r.Bottom() {
			break
		}
		x := r.Pos.X
		if slack := r.Size.W - strWidth(ln); slack > 0 {
			switch t.Align {
			case AlignCenter:
				x += slack / 2
			case AlignRight:
				x += slack
			}
		}
		ctx.Screen.Print(x, y, r.Right(), ln, t.Style)
	}
}

// lines produces the strings to render for a rect of the given width.
func (t *Text) lines(width int) []string {
	if !t.Wrap {
		return strings.Split(t.Content, "\n")
	}
	return WrapText(t.Content, width)
}

// WrapText greedily wraps s to the given display width (per cell.RuneWidth).
//
// Explicit newlines start new lines, whitespace runs collapse to single
// separators, and words wider than width are hard-broken at width
// boundaries. width <= 0 returns s as the sole line. The result always
// contains at least one line.
func WrapText(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	var out []string
	cur, curW := "", 0
	flush := func() { out = append(out, cur); cur, curW = "", 0 }
	for _, para := range strings.Split(s, "\n") {
		cur, curW = "", 0
		for _, word := range strings.Fields(para) {
			if ww := strWidth(word); ww > width {
				// Oversized word: emit what is on the line, then whole
				// chunks, leaving the tail open for following words.
				if cur != "" {
					flush()
				}
				chunks, tail := splitWide(word, width)
				out = append(out, chunks...)
				cur, curW = tail, strWidth(tail)
				continue
			}
			switch {
			case curW == 0:
				cur, curW = word, strWidth(word)
			case curW+1+strWidth(word) <= width:
				cur += " " + word
				curW += 1 + strWidth(word)
			default:
				flush()
				cur, curW = word, strWidth(word)
			}
		}
		out = append(out, cur)
	}
	return out
}

// splitWide cuts s into chunks of at most width display columns, returning
// full chunks plus the tail. A rune wider than width lands in its own chunk;
// such a chunk necessarily overflows and Print clips it.
func splitWide(s string, width int) (chunks []string, tail string) {
	var b strings.Builder
	w := 0
	for _, r := range s {
		rw := cell.RuneWidth(r)
		if w > 0 && w+rw > width {
			chunks = append(chunks, b.String())
			b.Reset()
			w = 0
		}
		b.WriteRune(r)
		w += rw
	}
	return chunks, b.String()
}

// Truncate cuts s to the given display width, appending tail when anything
// was removed so the result never exceeds width. width <= 0 yields "". A
// tail wider than width is dropped rather than breaking the budget.
func Truncate(s string, width int, tail string) string {
	if width <= 0 {
		return ""
	}
	if strWidth(s) <= width {
		return s
	}
	limit := width - strWidth(tail)
	withTail := limit >= 0
	if !withTail {
		limit = width
	}
	var b strings.Builder
	w := 0
	for _, r := range s {
		rw := cell.RuneWidth(r)
		if w+rw > limit {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	if withTail {
		b.WriteString(tail)
	}
	return b.String()
}

// strWidth sums cell.RuneWidth over every rune of s.
func strWidth(s string) int {
	w := 0
	for _, r := range s {
		w += cell.RuneWidth(r)
	}
	return w
}

// fit clamps a preferred size down to max. Dimensions <= 0 in max mean
// unconstrained — the convention Measure uses to ask "what do you want?".
func fit(pref, max geom.Size) geom.Size {
	if max.W > 0 && pref.W > max.W {
		pref.W = max.W
	}
	if max.H > 0 && pref.H > max.H {
		pref.H = max.H
	}
	return pref
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
