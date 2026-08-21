package widget

import (
	"fmt"
	"math"

	"github.com/Aswanidev-vs/cherry/cell"
	"github.com/Aswanidev-vs/cherry/geom"
)

// Progress renders Value as a bar of fill and empty cells, optionally
// followed by a right-aligned percentage label. It draws only the top row
// of the assigned rect.
type Progress struct {
	Base

	Value       float64    // read as clamped to [0, 1]
	FillRune    rune       // zero falls back to '█'
	EmptyRune   rune       // zero falls back to '░'
	ShowPercent bool       // reserve a 4-cell " NN%" label at the right edge
	Style       cell.Style // fill cells and label
	EmptyStyle  cell.Style // empty cells; zero keeps terminal default
}

func (p *Progress) Measure(max geom.Size) geom.Size {
	w := 1
	if p.ShowPercent {
		w += len(" NN%") // the fixed-width percent label
	}
	return fit(geom.Size{W: w, H: 1}, max)
}

func (p *Progress) Draw(ctx *DrawCtx) {
	r := ctx.Rect
	if r.Empty() {
		return
	}
	label := ""
	if p.ShowPercent {
		label = fmt.Sprintf("%3d%%", int(math.Round(p.ratio()*100)))
	}
	barW := r.Size.W - strWidth(label)
	if barW <= 0 {
		return
	}
	fill, empty := p.runes()
	filled := filledCells(barW, p.ratio())
	c := cell.Cell{Width: 1}
	for i := 0; i < barW; i++ {
		if i < filled {
			c.Rune, c.Style = fill, p.Style
		} else {
			c.Rune, c.Style = empty, p.EmptyStyle
		}
		ctx.Screen.Set(r.Pos.X+i, r.Pos.Y, c)
	}
	if label != "" {
		ctx.Screen.Print(r.Right()-strWidth(label), r.Pos.Y, r.Right(), label, p.Style)
	}
}

func (p *Progress) ratio() float64 {
	switch {
	case p.Value < 0:
		return 0
	case p.Value > 1:
		return 1
	default:
		return p.Value
	}
}

// filledCells reports how many of barW cells the fill occupies. Floor keeps
// the bar honest: it only fills what the value fully earns.
func filledCells(barW int, ratio float64) int {
	return int(math.Floor(float64(barW) * ratio))
}

func (p *Progress) runes() (fill, empty rune) {
	if p.FillRune == 0 {
		fill = '█'
	} else {
		fill = p.FillRune
	}
	if p.EmptyRune == 0 {
		empty = '░'
	} else {
		empty = p.EmptyRune
	}
	return fill, empty
}
