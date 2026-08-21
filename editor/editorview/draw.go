package editorview

import (
	"fmt"
	"unicode/utf8"

	"github.com/Aswanidev-vs/cherry/cell"
	"github.com/Aswanidev-vs/cherry/widget"

	"github.com/Aswanidev-vs/panda-editor/editor/document"
	"github.com/Aswanidev-vs/panda-editor/editor/highlight"
)

const defaultTabWidth = 4

var (
	gutterStyle = cell.Plain.Foreground(cell.Indexed(240))
	lineBgStyle = cell.Plain.Background(cell.Indexed(237))
)

type cachedSpan struct {
	text  string
	style cell.Style
}

// lexRow caches one line's lex result; valid only while line and entry
// state still match the buffer.
type lexRow struct {
	lexed  bool
	line   string
	in     highlight.State
	out    highlight.State
	nrunes int
	spans  []cachedSpan
}

// seg is one drawable rune at absolute screen column x.
type seg struct {
	r     rune
	x, w  int // w: display width (tabs and wide runes span several columns)
	ri    int // rune index inside the line
	style cell.Style
}

// Draw paints the viewport owned by ctx.Rect: a right-aligned line-number
// gutter, one margin column, then the text area with syntax spans, tab
// expansion, current-line highlight and selection inversion.
func (v *View) Draw(ctx *widget.DrawCtx) {
	doc := v.doc
	if doc == nil || ctx == nil || ctx.Screen == nil || ctx.Rect.Empty() {
		return
	}
	buf := doc.Buffer()
	lineCount := doc.LineCount()
	rect := ctx.Rect
	gw := digitCount(lineCount) + 1
	gx := rect.Pos.X
	textX := gx + gw + 1
	right := rect.Right()

	selOK := false
	var sel0, sel1 document.Pos
	if s, e, ok := doc.Selection(); ok {
		selOK, sel0, sel1 = true, s, e
	}

	v.growRows(lineCount)
	bottom := doc.ScrollY() + rect.Size.H
	if bottom > lineCount {
		bottom = lineCount
	}
	states := v.chainTo(doc, bottom)
	cur := doc.Cursor()

	for y := 0; y < rect.Size.H; y++ {
		lineNo := doc.ScrollY() + y
		sy := rect.Pos.Y + y
		if lineNo >= lineCount {
			fillCells(ctx, gx, right, sy, ' ', cell.Plain)
			continue
		}
		row := &v.rows[lineNo]
		line := buf.Line(lineNo)
		if !row.lexed || row.line != line || row.in != states[lineNo] {
			v.lexRow(line, states[lineNo], row)
		}
		drawGutter(ctx, gx, gw, sy, lineNo)
		if v.focused && !selOK && cur.Line == lineNo {
			fillCells(ctx, textX, right, sy, ' ', lineBgStyle)
		}
		segs := buildSegs(row, textX, right, v.tabWidth)
		paintSegs(ctx, sy, segs)
		if selOK {
			drawSelection(ctx, sy, segs, row, lineNo, sel0, sel1)
		}
	}
	v.lastRect = ctx.Rect
	v.drawn = true
}

// chainTo guarantees the highlight entry states [0,n] with a forward walk
// that re-lexes only rows whose text or entry state drifted.
func (v *View) chainTo(doc *document.Document, n int) []highlight.State {
	buf := doc.Buffer()
	if len(v.states) < n+1 {
		ns := make([]highlight.State, n+1)
		copy(ns, v.states)
		v.states = ns
	} else {
		v.states = v.states[:n+1]
	}
	for k := 0; k < n; k++ {
		row := &v.rows[k]
		line := buf.Line(k)
		if row.lexed && row.line == line && row.in == v.states[k] {
			continue
		}
		v.lexRow(line, v.states[k], row)
		v.states[k+1] = row.out
	}
	return v.states
}

func (v *View) growRows(n int) {
	if len(v.rows) >= n {
		return
	}
	rows := make([]lexRow, n)
	copy(rows, v.rows)
	v.rows = rows
}

func (v *View) lexRow(line string, in highlight.State, row *lexRow) {
	row.line = line
	row.in = in
	row.nrunes = utf8.RuneCountInString(line)
	if len(line) == 0 {
		row.lexed = true
		row.out = in
		row.spans = nil
		return
	}
	hspans, out := v.lexer.Style(line, in)
	row.lexed = true
	row.out = out
	row.spans = row.spans[:0]
	for _, s := range hspans {
		row.spans = append(row.spans, cachedSpan{text: s.Text, style: s.Style})
	}
}

func digitCount(n int) int {
	d := 1
	for n >= 10 {
		n /= 10
		d++
	}
	return d
}

func drawGutter(ctx *widget.DrawCtx, gx, gw, sy, lineNo int) {
	fillCells(ctx, gx, gx+gw, sy, ' ', gutterStyle)
	max := gw - 1
	label := fmt.Sprintf("%d", lineNo+1)
	if len(label) > max {
		label = label[len(label)-max:]
	}
	ctx.Screen.Print(gx+(max-len(label)), sy, gx+max, label, gutterStyle)
}

func fillCells(ctx *widget.DrawCtx, x0, x1, y int, r rune, st cell.Style) {
	for x := x0; x < x1; x++ {
		ctx.Screen.Set(x, y, cell.Cell{Rune: r, Style: st, Width: 1})
	}
}

func nextTabStop(x, textX, tabWidth int) int {
	if tabWidth < 1 {
		tabWidth = 1
	}
	return x + tabWidth - (x-textX)%tabWidth
}

// buildSegs walks the line's spans rune by rune like Print would: tabs fill
// to the next stop, zero-width runes occupy nothing, wide runes take two
// columns and a wide rune straddling the right edge stops the line.
func buildSegs(row *lexRow, textX, right, tabWidth int) []seg {
	segs := []seg{}
	x, ri := textX, 0
	for _, sp := range row.spans {
		for _, r := range sp.text {
			if x >= right {
				break
			}
			w := cell.RuneWidth(r)
			switch {
			case r == '\t':
				stop := nextTabStop(x, textX, tabWidth)
				if stop > right {
					stop = right
				}
				segs = append(segs, seg{r: r, x: x, w: stop - x, ri: ri, style: sp.style})
				x = stop
			case w == 0:
			case w == 1:
				segs = append(segs, seg{r: r, x: x, w: 1, ri: ri, style: sp.style})
				x++
			case x+1 < right:
				segs = append(segs, seg{r: r, x: x, w: 2, ri: ri, style: sp.style})
				x += 2
			default:
				x = right
			}
			ri++
		}
		if x >= right {
			break
		}
	}
	return segs
}

func paintSegs(ctx *widget.DrawCtx, sy int, segs []seg) {
	scr := ctx.Screen
	for _, s := range segs {
		if s.r == '\t' {
			st := s.style
			for i := 0; i < s.w; i++ {
				scr.Set(s.x+i, sy, cell.Cell{Rune: ' ', Style: st, Width: 1})
			}
			continue
		}
		scr.Set(s.x, sy, cell.Cell{Rune: s.r, Style: s.style, Width: uint8(s.w)})
		if s.w == 2 {
			scr.Set(s.x+1, sy, cell.Cell{Rune: ' ', Style: s.style, Width: 0})
		}
	}
}

func lineSelRange(lineNo, nrunes int, sel0, sel1 document.Pos) (from, to int) {
	if lineNo < sel0.Line || lineNo > sel1.Line {
		return 0, 0
	}
	from, to = 0, nrunes
	if lineNo == sel0.Line {
		from = sel0.Col
	}
	if lineNo == sel1.Line {
		to = sel1.Col
	}
	if from > nrunes {
		from = nrunes
	}
	if to > nrunes {
		to = nrunes
	}
	if from >= to {
		return 0, 0
	}
	return from, to
}

// drawSelection inverts the style of every cell whose rune index on this
// line falls inside the ordered selection endpoints.
func drawSelection(ctx *widget.DrawCtx, sy int, segs []seg, row *lexRow, lineNo int, sel0, sel1 document.Pos) {
	from, to := lineSelRange(lineNo, row.nrunes, sel0, sel1)
	if from >= to {
		return
	}
	scr := ctx.Screen
	for _, s := range segs {
		if s.ri < from || s.ri >= to {
			continue
		}
		st := s.style.Reverse(true)
		for i := 0; i < s.w; i++ {
			c := cell.Cell{Style: st, Width: 1}
			switch {
			case s.r == '\t':
				c.Rune = ' '
			case i == 0:
				c.Rune = s.r
				c.Width = uint8(s.w)
			default:
				c.Rune = ' '
				c.Width = 0
			}
			scr.Set(s.x+i, sy, c)
		}
	}
}

// runWidthTo returns the absolute x the cursor occupies at rune col of the
// line, applying the same tab/width rules as buildSegs.
func runWidthTo(line string, col, startX, tabWidth int) int {
	x := startX
	i := 0
	for _, r := range line {
		if i == col {
			break
		}
		if r == '\t' {
			x = nextTabStop(x, startX, tabWidth)
		} else if w := cell.RuneWidth(r); w > 0 {
			x += w
		}
		i++
	}
	return x
}
