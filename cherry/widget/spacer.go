package widget

import (
	"github.com/Aswanidev-vs/cherry/cell"
	"github.com/Aswanidev-vs/cherry/geom"
)

// Spacer reserves blank space. With FillX/FillY it absorbs whatever surplus
// the layout solver offers, up to the constraint it is measured with;
// otherwise it holds MinW x MinH.
type Spacer struct {
	Base

	MinW  int
	MinH  int
	FillX bool
	FillY bool
	Bg    cell.Style // optional fill; zero value clears to default style
}

func (s *Spacer) Measure(max geom.Size) geom.Size {
	size := geom.Size{W: s.MinW, H: s.MinH}
	if s.FillX && max.W > size.W {
		size.W = max.W
	}
	if s.FillY && max.H > size.H {
		size.H = max.H
	}
	return fit(size, max)
}

func (s *Spacer) Draw(ctx *DrawCtx) {
	if !ctx.Rect.Empty() {
		ctx.Screen.Fill(ctx.Rect, styledBlank(s.Bg))
	}
}
