package widget

import (
	"github.com/Aswanidev-vs/cherry/cell"
	"github.com/Aswanidev-vs/cherry/geom"
	"github.com/Aswanidev-vs/cherry/input"
)

// BorderMode selects the frame a Box draws around its child.
type BorderMode uint8

const (
	BorderNone    BorderMode = iota
	BorderSingle
	BorderRounded
	BorderDouble
)

// borderRunes holds the six glyphs a frame needs:
// top-left, top-right, bottom-left, bottom-right, horizontal, vertical.
type borderRunes [6]rune

var borders = map[BorderMode]borderRunes{
	BorderSingle:  {'┌', '┐', '└', '┘', '─', '│'},
	BorderRounded: {'╭', '╮', '╰', '╯', '─', '│'},
	BorderDouble:  {'╔', '╗', '╚', '╝', '═', '║'},
}

// Box is a padded, optionally bordered container for one optional Child.
// The whole rect is first filled with Background so the interior paints
// over whatever the previous frame left behind.
//
// Handle forwards events to Child: positional (mouse) events only when the
// point lies inside the region assigned during the last Draw, all others
// unconditionally.
type Box struct {
	Mode        BorderMode
	Title       string     // drawn into the top border with one space of padding
	PadT        int        // padding above the child
	PadR        int        // padding right of the child
	PadB        int        // padding below the child
	PadL        int        // padding left of the child
	Background  cell.Style // fill applied to the full rect before anything else draws
	BorderStyle cell.Style // style for border runes and the title
	Child       Widget

	lastChild geom.Rect // inner region handed to Child during the last Draw
}

func (b *Box) Measure(max geom.Size) geom.Size {
	fl, ft := b.frame()
	hc, vc := 2*fl+b.PadL+b.PadR, 2*ft+b.PadT+b.PadB
	var pref geom.Size
	if b.Child != nil {
		// Chrome is subtracted from positive constraints only; a dimension
		// of <= 0 means unconstrained (see fit), so it passes through as-is.
		var inner geom.Size
		if max.W > 0 {
			inner.W = maxInt(max.W-hc, 0)
		}
		if max.H > 0 {
			inner.H = maxInt(max.H-vc, 0)
		}
		pref = b.Child.Measure(inner)
	} else if b.Title != "" {
		pref.W = strWidth(" " + b.Title + " ")
	}
	return fit(geom.Size{W: pref.W + hc, H: pref.H + vc}, max)
}

func (b *Box) Draw(ctx *DrawCtx) {
	b.lastChild = geom.Rect{}
	r := ctx.Rect
	if r.Empty() {
		return
	}
	ctx.Screen.Fill(r, styledBlank(b.Background))

	fl, ft := b.frame()
	if fl > 0 && r.Size.W >= 2 && r.Size.H >= 2 {
		b.drawFrame(ctx)
		if b.Title != "" && r.Size.W >= 5 {
			b.drawTitle(ctx)
		}
	}

	iw, ih := r.Size.W-2*fl-b.PadL-b.PadR, r.Size.H-2*ft-b.PadT-b.PadB
	if b.Child == nil || iw <= 0 || ih <= 0 {
		return
	}
	b.lastChild = geom.Rect{
		Pos:  geom.Point{X: r.Pos.X + fl + b.PadL, Y: r.Pos.Y + ft + b.PadT},
		Size: geom.Size{W: iw, H: ih},
	}
	b.Child.Draw(&DrawCtx{Rect: b.lastChild, Screen: ctx.Screen})
}

func (b *Box) Handle(ev input.Event) bool {
	if b.Child == nil {
		return false
	}
	if m, ok := ev.(input.Mouse); ok && !b.lastChild.Contains(geom.Point{X: m.X, Y: m.Y}) {
		return false
	}
	return b.Child.Handle(ev)
}

// frame reports how many cells the border occupies on each axis.
func (b *Box) frame() (left, top int) {
	if b.Mode == BorderNone {
		return 0, 0
	}
	return 1, 1
}

func (b *Box) drawFrame(ctx *DrawCtx) {
	br, ok := borders[b.Mode]
	if !ok {
		return
	}
	r := ctx.Rect
	x0, x1 := r.Pos.X, r.Right()-1
	y0, y1 := r.Pos.Y, r.Bottom()-1
	c := styledBlank(b.BorderStyle)
	set := func(x, y int, ru rune) { c.Rune = ru; ctx.Screen.Set(x, y, c) }
	for x := x0 + 1; x < x1; x++ {
		set(x, y0, br[4])
		set(x, y1, br[4])
	}
	for y := y0 + 1; y < y1; y++ {
		set(x0, y, br[5])
		set(x1, y, br[5])
	}
	set(x0, y0, br[0])
	set(x1, y0, br[1])
	set(x0, y1, br[2])
	set(x1, y1, br[3])
}

// drawTitle overlays " Title " onto the top border. The rect-width guard in
// Draw keeps two leading columns and two trailing columns of frame intact.
func (b *Box) drawTitle(ctx *DrawCtx) {
	r := ctx.Rect
	text := Truncate(" "+b.Title+" ", r.Size.W-4, "…")
	if text == "" {
		return
	}
	ctx.Screen.Print(r.Pos.X+2, r.Pos.Y, r.Right()-1, text, b.BorderStyle)
}

// styledBlank is a space cell painted with st, used for background fills.
func styledBlank(st cell.Style) cell.Cell { return cell.Cell{Rune: ' ', Style: st, Width: 1} }
